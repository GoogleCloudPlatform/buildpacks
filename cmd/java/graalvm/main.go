from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import argparse
import asyncio
from typing import Dict, Any
import lib.detect as detect_lib
import lib.build as build_lib

app = FastAPI()

class DetectRequest(BaseModel):
    app: Dict[str, Any]
    environment: Dict[str, str]

class DetectResponse(BaseModel):
    GraalVMVersion: str
    NativeImageEnabled: bool

class BuildRequest(BaseModel):
    app: Dict[str, Any]
    environment: Dict[str, str]
    buildpack_plan: Dict[str, Any]

class BuildResponse(BaseModel):
    success: bool
    message: str

@app.post("/detect")
async def detect(request: DetectRequest) -> DetectResponse:
    try:
        result = await detect_lib.detect_fn(request.app, request.environment)
        return DetectResponse(
            GraalVMVersion=result.get("GraalVMVersion", ""),
            NativeImageEnabled=bool(result.get("NativeImageEnabled", False))
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def build(request: BuildRequest) -> BuildResponse:
    try:
        success, message = await build_lib.build_fn(
            request.app,
            request.environment,
            request.buildpack_plan
        )
        return BuildResponse(success=success, message=message)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

async def main():
    parser = argparse.ArgumentParser(description='Java GraalVM Native Image Buildpack')
    parser.add_argument('--detect', action='store_true', help='Run detection logic')
    parser.add_argument('--build', action='store_true', help='Run build logic')
    args = parser.parse_args()

    if not (args.detect or args.build):
        print("Please specify either --detect or --build")
        return

    if args.detect:
        # Simulate detect request
        detect_request = DetectRequest(
            app={},
            environment={}
        )
        result = await detect_lib.detect_fn(detect_request.app, detect_request.environment)
        print(f"Detection Result: {result}")

    if args.build:
        # Simulate build request
        build_request = BuildRequest(
            app={},
            environment {},
            buildpack_plan={}
        )
        success, message = await build_lib.build_fn(build_request.app, build_request.environment, build_request.buildpack_plan)
        print(f"Build Result: Success={success}, Message='{message}'")

if __name__ == "__main__":
    asyncio.run(main())
