from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import shutil
import os

app = FastAPI()

class DetectRequest(BaseModel):
    environment: dict
    files: list[str]

class DetectResponse(BaseModel):
    applicable: bool
    priority: int

class BuildRequest(BaseModel):
    source_path: str
    output_path: str
    environment: dict

class BuildResponse(BaseModel):
    run_script_path: str

async def detect(request: DetectRequest) -> DetectResponse:
    """
    Detect if this buildpack applies to the current build context.
    """
    # Example detection logic (can be customized)
    has_firebase = "firebase.json" in request.files
    return DetectResponse(applicable=has_firebase, priority=0)

async def copy_assets(source_path: str, output_path: str) -> None:
    """Copy static assets to the output directory."""
    try:
        # Use asyncio.to_thread to run blocking operations in separate threads
        await asyncio.to_thread(
            shutil.copytree,
            source_path,
            output_path,
            ignore=shutil.ignore_patterns('node_modules', '*.pyc')
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error copying assets: {str(e)}")

async def override_run_script(output_path: str) -> str:
    """Generate and return the path to the new run script."""
    run_script = os.path.join(output_path, "run")
    try:
        await asyncio.to_thread(
            lambda: open(run_script, 'w').write("#!/bin/sh\nnode dist/main.js")
        )
        # Make the script executable
        await asyncio.to_thread(lambda: os.chmod(run_script, 0o755))
        return run_script
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error creating run script: {str(e)}")

@app.post("/detect")
async def handle_detect(request: DetectRequest) -> DetectResponse:
    """Handle detect request."""
    return await detect(request)

@app.post("/build")
async def handle_build(request: BuildRequest) -> BuildResponse:
    """Handle build request."""
    try:
        # Create output directory if it doesn't exist
        await asyncio.to_thread(lambda: os.makedirs(request.output_path, exist_ok=True))

        # Copy static assets
        await copy_assets(request.source_path, request.output_path)

        # Generate run script
        run_script_path = await override_run_script(request.output_path)

        return BuildResponse(run_script_path=run_script_path)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn

    # Configure CORS
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    uvicorn.run(app, host="0.0.0.0", port=8080)
