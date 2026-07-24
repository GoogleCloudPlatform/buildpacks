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

from fastapi import FastAPI
from pydantic import BaseModel
import asyncio
import os
from pathlib import Path

app = FastAPI()

class JavaRuntimeSettings(BaseModel):
    version: str = "17"
    distribution: str = "temurin"

class DetectRequest(BaseModel):
    app_dir: str
    cache_dir: str
    env: dict[str, str]
    settings: dict[str, str]

class DetectResponse(BaseModel):
    detected: bool
    id: str
    version: str
    settings: JavaRuntimeSettings

class BuildRequest(BaseModel):
    app_dir: str
    cache_dir: str
    env: dict[str, str]
    settings: JavaRuntimeSettings

class BuildResponse(BaseModel):
    success: bool
    message: str

async def detect(request: DetectRequest) -> DetectResponse:
    app_path = Path(request.app_dir)
    if (app_path / "pom.xml").exists():
        return DetectResponse(
            detected=True,
            id="java-runtime",
            version=request.settings.get("version", "17"),
            settings=JavaRuntimeSettings()
        )
    return DetectResponse(detected=False)

async def build(request: BuildRequest) -> BuildResponse:
    try:
        # Install JDK
        await asyncio.create_subprocess_exec(
            "apt-get", "update",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        await asyncio.create_subprocess_exec(
            "apt-get", "-y", "install", "openjdk-17-jdk",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )

        # Set JAVA_HOME
        java_home = Path("/usr/lib/jvm/java-17-openjdk-amd64")
        os.environ["JAVA_HOME"] = str(java_home.resolve())

        return BuildResponse(success=True, message="Java runtime installed successfully")
    except Exception as e:
        return BuildResponse(success=False, message=str(e))

@app.post("/detect")
async def detect_endpoint(request: DetectRequest) -> DetectResponse:
    return await detect(request)

@app.post("/build")
async def build_endpoint(request: BuildRequest) -> BuildResponse:
    return await build(request)

async def main():
    try:
        import uvicorn
        from fastapi.middleware.cors import CORSMiddleware

        app.add_middleware(
            CORSMiddleware,
            allow_origins=["*"],
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"]
        )

        await asyncio.create_subprocess_exec(
            "uvicorn",
            "--host", "localhost",
            "--port", "8080",
            "--app-dir", ".",
            "main:app"
        )
    except Exception as e:
        print(f"Error starting server: {e}")
        raise

if __name__ == "__main__":
    asyncio.run(main())
