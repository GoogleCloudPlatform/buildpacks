# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import argparse
import subprocess
import os

app = FastAPI()

class BuildRequest(BaseModel):
    project_path: str
    output_dir: str | None = None

async def detect_fn(project_path: str) -> bool:
    """Detects if the current directory contains a .NET project."""
    try:
        # Check for common .NET project files
        return os.path.exists(os.path.join(project_path, "*.csproj"))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Detection error: {str(e)}")

async def build_fn(request: BuildRequest) -> dict:
    """Runs dotnet publish command."""
    try:
        project_path = request.project_path
        output_dir = request.output_dir

        # Run dotnet publish with async subprocess
        cmd = ["dotnet", "publish"]
        if output_dir:
            cmd.extend(["-o", output_dir])
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            cwd=project_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        stdout, stderr = await proc.communicate()

        if proc.returncode != 0:
            raise HTTPException(
                status_code=500,
                detail=f"Publish failed: {stderr.decode()}"
            )

        return {
            "status": "success",
            "output": stdout.decode(),
            "error": stderr.decode(),
            "returncode": proc.returncode
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Build error: {str(e)}")

@app.post("/detect")
async def detect_endpoint(request: BuildRequest) -> dict:
    result = await detect_fn(request.project_path)
    return {"detected": result}

@app.post("/build")
async def build_endpoint(request: BuildRequest) -> dict:
    return await build_fn(request)

def main():
    parser = argparse.ArgumentParser(description='dotnet publish buildpack')
    parser.add_argument('--detect', action='store_true',
                       help='Check if current directory contains a .NET project.')
    parser.add_argument('--build', action='store_true',
                       help='Run dotnet publish.')
    parser.add_argument('--project-path', type=str,
                       help='Path to the .NET project directory.')
    parser.add_argument('--output-dir', type=str, default=None,
                       help='Output directory for published files.')
    parser.add_argument('--port', type=int, default=8000,
                       help='Port to run the FastAPI server on.')

    args = parser.parse_args()

    if not (args.detect or args.build):
        # Start FastAPI server
        import uvicorn
        uvicorn.run(app, host="0.0.0.0", port=args.port)
        return

    project_path = args.project_path or os.getcwd()

    try:
        if args.detect:
            detected = asyncio.run(detect_fn(project_path))
            print(f"Detected .NET project: {detected}")

        if args.build:
            output = asyncio.run(build_fn(BuildRequest(
                project_path=project_path,
                output_dir=args.output_dir
            )))
            print("Build completed successfully")
            print("Output:", output["output"])
            print("Error:", output["error"])
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    main()
