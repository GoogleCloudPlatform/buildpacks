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

import os
import subprocess
from pathlib import Path
from typing import Dict

import asyncio
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from starlette.middleware.cors import CORSMiddleware

app = FastAPI()

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class DetectResponse(BaseModel):
    match: bool

class BuildResponse(BaseModel):
    output: str | None = None
    error: str | None = None

@app.get("/detect", response_model=DetectResponse)
async def detect():
    try:
        # Check if composer.json and composer.lock exist
        composer_json = Path("composer.json").resolve()
        composer_lock = Path("composer.lock").resolve()

        match = (await asyncio.to_thread(composer_json.exists) and
                await asyncio.to_thread(composer_lock.exists))

        return {"match": match}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build", response_model=BuildResponse)
async def build():
    try:
        # Run gcp-build script
        result = await asyncio.get_event_loop().run_in_executor(
            None,
            lambda: subprocess.run(
                ["composer", "gcp-build"],
                capture_output=True,
                text=True,
                check=True
            )
        )

        return {"output": result.stdout}
    except subprocess.CalledProcessError as e:
        return {"error": str(e)}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    # Set environment variables
    os.environ["PYTHONPATH"] = ":".join([
        "lib/python3/dist-packages",
        "lib64/python3/dist-packages"
    ])

    asyncio.run(app.serve(host="0.0.0.0", port=8080))
