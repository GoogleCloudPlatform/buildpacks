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

from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel
import asyncio
import json
import os
import subprocess
import logging

app = FastAPI()

class AppInfo(BaseModel):
    app_type: str

class BuildResult(BaseModel):
    success: bool
    message: str | None
    build_output: dict | None

async def detect_buildpack():
    # Check for .NET project files
    if os.path.exists("project.json") or any(fname.endswith((".csproj", ".sln")) for fname in os.listdir()):
        return AppInfo(app_type="dotnet-flex")
    else:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Not a .NET Flex application")

async def build_dotnet():
    try:
        # Set GAE environment variables
        env_vars = {
            "GAE_ENV": "flex",
            "ASPNETCORE_URLS": "http://+:8080"
        }

        # Run dotnet publish with GCP settings
        cmd = ["dotnet", "publish", "-c", "Release", "-o", "bin/Debug/net6.0/publish"]
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env={**os.environ, **env_vars}
        )

        stdout, stderr = await proc.communicate()

        if proc.returncode != 0:
            raise Exception(f"Build failed: {stderr.decode()}")

        return BuildResult(
            success=True,
            message="Build successful",
            build_output=json.loads(stdout.decode())
        )
    except Exception as e:
        logging.error(f"Build error: {str(e)}")
        return BuildResult(success=False, message=str(e), build_output=None)

@app.get("/detect")
async def detect():
    return await detect_buildpack()

@app.post("/build")
async def build(app_info: AppInfo):
    return await build_dotnet()

if __name__ == "__main__":
    import uvicorn
    from fastapi.middleware.cors import CORSMiddleware

    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"]
    )

    uvicorn.run(app, host="0.0.0.0", port=8000)
