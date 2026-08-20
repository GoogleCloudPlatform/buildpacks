from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio

app = FastAPI(
    title="Dotnet Buildpacks Runner",
    description="Runs .NET language builder buildpacks for Google Cloud Platform",
    version="1.0.0"
)

class DetectRequest(BaseModel):
    buildpack_id: str

class DetectResponse(BaseModel):
    detected: bool
    message: str

class BuildRequest(BaseModel):
    buildpack_id: str
    # Add any other required parameters here

class BuildResponse(BaseModel):
    success: bool
    output: dict

# Register buildpack functions here
buildpacks = {}

async def detect_handler(request: DetectRequest):
    try:
        buildpack = buildpacks.get(request.buildpack_id)
        if not buildpack or not buildpack["detect"]:
            raise HTTPException(status_code=404, detail="Buildpack not found")

        # Run detection logic
        detected = await buildpack["detect"]()
        return {"detected": detected, "message": "Detection completed successfully"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

async def build_handler(request: BuildRequest):
    try:
        buildpack = buildpacks.get(request.buildpack_id)
        if not buildpack or not buildpack["build"]:
            raise HTTPException(status_code=404, detail="Buildpack not found")

        # Run build logic
        result = await buildpack["build"](request.dict())
        return {"success": True, "output": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/")
async def root():
    return {"message": "Dotnet Buildpacks Runner"}

@app.post("/detect")
async def detect(request: DetectRequest):
    return await detect_handler(request)

@app.post("/build")
async def build(request: BuildRequest):
    return await build_handler(request)
