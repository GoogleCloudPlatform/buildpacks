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

import asyncio
import os
from pathlib import Path

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class BuildpackRequest(BaseModel):
    pubspec_path: str = "pubspec.yaml"

@app.on_event("startup")
async def startup():
    pass  # Initialize if needed

async def detect(pubspec_path: str) -> bool:
    """Detects if the pub buildpack is needed based on pubspec.yaml presence."""
    try:
        return os.path.exists(pubspec_path)
    except Exception as e:
        print(f"Error detecting pubspec file: {e}")
        return False

async def run_pub_command(command: str) -> None:
    """Runs a pub command asynchronously."""
    process = await asyncio.create_subprocess_exec(
        'pub', *command.split(),
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )
    stdout, stderr = await process.communicate()
    if process.returncode != 0:
        raise RuntimeError(f"Pub command failed: {stderr.decode()}")
    print(f"Pub command output:\n{stdout.decode()}")

@app.post("/build")
async def build_endpoint(request: BuildpackRequest) -> dict:
    """Handles the build request asynchronously."""
    try:
        # Detect phase
        pubspec_path = Path(request.pubspec_path)
        if not await asyncio.to_thread(pubspec_path.exists):
            return {"message": "No pubspec.yaml found, skipping build."}

        # Build phase
        print("Installing dependencies with pub...")
        await run_pub_command("get")
        await run_pub_command("install")

        return {"message": "Build completed successfully."}
    except Exception as e:
        return {"error": str(e)}

if __name__ == "__main__":
    asyncio.run(build_endpoint(BuildpackRequest()))
