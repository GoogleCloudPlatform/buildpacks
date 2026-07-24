"""
Copyright 2025 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import os
import tarfile
import aiofiles
import hashlib

app = FastAPI()

class BuildpackSettings(BaseModel):
    app_dir: str = "/workspace"
    cache_dir: str = ".cache"

class DetectRequest(BaseModel):
    buildpack_id: str

class DetectResponse(BaseModel):
    detected: bool
    message: str

class BuildRequest(BaseModel):
    buildpack_id: str
    settings: BuildpackSettings

async def detect(buildpack_id: str) -> DetectResponse:
    """
    Detects if the app requires Ruby runtime based on presence of Gemfile.
    """
    try:
        gemfile_path = os.path.join(os.getenv("APP_DIR", "/workspace"), "Gemfile")
        async with aiofiles.open(gemfile_path, mode='r') as f:
            content = await f.read()
            if content.strip():
                return DetectResponse(detected=True, message="Ruby runtime detected via Gemfile.")
    except FileNotFoundError:
        pass

    # Check for other Ruby-related files
    try:
        ruby_files = ["Rakefile", "config.ru"]
        for file in ruby_files:
            file_path = os.path.join(os.getenv("APP_DIR", "/workspace"), file)
            async with aiofiles.open(file_path, mode='r') as f:
                content = await f.read()
                if content.strip():
                    return DetectResponse(detected=True, message=f"Ruby runtime detected via {file}.")
    except FileNotFoundError:
        pass

    return DetectResponse(detected=False, message="No Ruby runtime requirements detected.")

async def build(buildpack_id: str, settings: BuildpackSettings) -> dict:
    """
    Installs the Ruby runtime.
    """
    try:
        ruby_version = "3.2.2"

        # Download and install Ruby
        package_url = f"https://cache.ruby-lang.org/pub/ruby/{ruby_version}/ruby-{ruby_version}.tar.gz"
        download_path = os.path.join(settings.cache_dir, f"ruby-{ruby_version}.tar.gz")

        async with aiofiles.open(download_path, mode='wb') as f:
            async with aiohttp.ClientSession() as session:
                async with session.get(package_url) as response:
                    if response.status == 200:
                        await f.write(await response.read())
                    else:
                        raise HTTPException(status_code=400, detail=f"Failed to download Ruby {ruby_version}")

        # Verify checksum
        expected_checksum = "expected-sha256-checksum"
        actual_checksum = hashlib.sha256()
        async with aiofiles.open(download_path, mode='rb') as f:
            while chunk := await f.read(8192):
                actual_checksum.update(chunk)

        if actual_checksum.hexdigest() != expected_checksum:
            raise HTTPException(status_code=400, detail=f"Checksum mismatch for Ruby {ruby_version}")

        # Extract and install
        with tarfile.open(download_path) as tar:
            extract_dir = os.path.join(settings.cache_dir, "ruby")
            tar.extractall(path=extract_dir)

        # Move to final location
        install_dir = "/usr/local"
        if not os.path.exists(install_dir):
            os.makedirs(install_dir)

        os.rename(os.path.join(extract_dir, f"ruby-{ruby_version}"), os.path.join(install_dir, "ruby"))

        return {"status": "success", "message": f"Ruby {ruby_version} installed successfully."}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error installing Ruby runtime: {str(e)}")

@app.post("/detect")
async def detect_endpoint(request: DetectRequest):
    result = await detect(request.buildpack_id)
    return result.dict()

@app.post("/build")
async def build_endpoint(request: BuildRequest):
    try:
        result = await build(request.buildpack_id, request.settings)
        return {"status": "success", **result}
    except HTTPException as e:
        raise e

@app.on_event("startup")
async def startup():
    print("Starting Ruby runtime buildpack service...")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
