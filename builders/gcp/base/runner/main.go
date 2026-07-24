import asyncio
from typing import Dict, Any
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI()

class BuildpackRequest(BaseModel):
    buildpack_id: str
    phase: str

class BuildpackResponse(BaseModel):
    output: str

# Import all buildpack functions
# Note: Ensure each function is properly exported from their respective modules
# Example:
# from buildpacks.google.cpp.clear_source import detect as cpp_clear_source_detect, build as cpp_clear_source_build
# ... and so on for all other buildpacks

buildpack_functions: Dict[str, Dict[str, asyncio.Future]] = {}

async def initialize_buildpacks():
    global buildpack_functions
    # Manually register each buildpack with their functions
    # Example:
    buildpack_functions["google.cpp.clear-source"] = {
        "detect": cpp_clear_source_detect,
        "build": cpp_clear_source_build
    }
    # ... continue for all other buildpacks

@app.on_event("startup")
async def startup_event():
    await initialize_buildpacks()

@app.post("/run_buildpack/")
async def run_buildpack(request: BuildpackRequest) -> BuildpackResponse:
    if request.buildpack_id not in buildpack_functions:
        raise HTTPException(status_code=404, detail="Buildpack ID not found")

    phase_func = None
    if request.phase == "detect":
        phase_func = buildpack_functions[request.buildpack_id]["detect"]
    elif request.phase == "build":
        phase_func = buildpack_functions[request.buildpack_id]["build"]
    else:
        raise HTTPException(status_code=400, detail="Invalid phase specified")

    try:
        result = await phase_func()
        return BuildpackResponse(output=result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

async def main():
    import uvicorn
    config = uvicorn.Config(app, host="0.0.0.0", port=8000)
    await config.start()

if __name__ == "__main__":
    asyncio.run(main())
