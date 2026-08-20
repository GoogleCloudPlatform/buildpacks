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
import os
import subprocess
from google_buildpacks import buildpack_main

app = FastAPI()

class ComposerConfig(BaseModel):
    composer_version: str = "2.5.8"
    composer_file_path: str = "composer.json"

class BuildpackConfig(BaseModel):
    platform: str
    runtime: str
    environment_variables: dict = {}

@app.post("/")
async def handle_composer_install(request_data: dict):
    try:
        # Extract necessary data from request
        platform = request_data.get("platform")
        runtime = request_data.get("runtime")
        composer_file_path = request_data.get("composer_file", "composer.json")

        if not os.path.exists(composer_file_path):
            raise HTTPException(status_code=400, detail="Composer file not found")

        # Run detection step
        detected = await asyncio.to_thread(
            buildpack_main.detect,
            platform=platform,
            runtime=runtime
        )

        if not detected:
            return {"status": "not_detected", "message": "Buildpack not detected"}

        # Perform build step
        built = await asyncio.to_thread(
            buildpack_main.build,
            composer_file=composer_file_path
        )

        if not built:
            raise HTTPException(status_code=500, detail="Build failed")

        return {"status": "success", "message": "Composer dependencies installed successfully"}

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

def main():
    import uvicorn
    from argparse import ArgumentParser

    parser = ArgumentParser()
    parser.add_argument("--host", type=str, default="0.0.0.0")
    parser.add_argument("--port", type=int, default=8080)
    args = parser.parse_args()

    uvicorn.run(app, host=args.host, port=args.port)

if __name__ == "__main__":
    main()
