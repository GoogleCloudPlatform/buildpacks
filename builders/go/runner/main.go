from fastapi import FastAPI, HTTPException, UploadFile, File, Form
from pydantic import BaseModel
import asyncio
import shutil
import os

app = FastAPI()

class BuildpackRequest(BaseModel):
    buildpack_id: str
    phase: str
    source_files: list[UploadFile]

class BuildpackResponse(BaseModel):
    output: dict[str, str]
    success: bool

# Register buildpack functions here
buildpacks = {}

async def detect(buildpack_id: str) -> bool:
    # Implement detection logic for each buildpack
    pass

async def build(buildpack_id: str, source_files: list[UploadFile]) -> dict:
    # Implement build logic for each buildpack
    pass

@app.post("/run_buildpack")
async def run_buildpack(request_data: BuildpackRequest) -> BuildpackResponse:
    try:
        if request_data.phase not in ["detect", "build"]:
            raise HTTPException(status_code=400, detail="Invalid phase specified")

        # Check if buildpack is registered
        if request_data.buildpack_id not in buildpacks:
            raise HTTPException(status_code=400, detail="Buildpack not found")

        # Run the appropriate phase
        if request_data.phase == "detect":
            result = await detect(request_data.buildpack_id)
        else:
            # Handle file uploads asynchronously
            upload_dir = f"uploads/{id(request_data)}"
            os.makedirs(upload_dir, exist_ok=True)

            for file in request_data.source_files:
                file_path = os.path.join(upload_dir, file.filename)
                with open(file_path, "wb") as buffer:
                    await asyncio.to_thread(shutil.copyfileobj, file.file, buffer)

            result = await build(request_data.buildpack_id, upload_dir)

        return BuildpackResponse(output=result, success=True)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

# Register your buildpacks here
def register_buildpack(buildpack_id: str):
    def decorator(detect_fn, build_fn):
        nonlocal buildpacks
        buildpacks[buildpack_id] = {
            "detect": detect_fn,
            "build": build_fn
        }
        return None
    return decorator

# Example usage:
@register_buildpack("google.go.runtime")
async def go_runtime_detect():
    # Implement detection logic
    pass

async def go_runtime_build(source_dir: str) -> dict:
    # Implement build logic
    pass

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
