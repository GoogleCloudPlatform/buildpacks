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

"""
Implements Go buildpack using FastAPI and Pydantic for data models.
This module handles detection and building of Go applications.
"""

import asyncio
from typing import Optional

import typer
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI()

# Local imports
import lib.detect
import lib.build
import gcp_buildpack

class BuildContext(BaseModel):
    """
    Represents the build context for Go applications.
    """
    app_path: str
    env_vars: Optional[dict] = None

class DetectionResult(BaseModel):
    """
    Result of the detection process.
    """
    applicable: bool
    errors: Optional[list[str]] = None
    warnings: Optional[list[str]] = None

@app.get("/")
async def root():
    """
    Root endpoint for testing service availability.
    """
    return {"message": "Go buildpack is running"}

@app.post("/detect")
async def detect(build_context: BuildContext) -> DetectionResult:
    """
    Detects if the Go buildpack should be applied to the current application.
    """
    try:
        result = await lib.detect.DetectFn(build_context)
        return DetectionResult(
            applicable=result.applicable,
            errors=result.errors,
            warnings=result.warnings
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def build(build_context: BuildContext) -> dict:
    """
    Builds the Go application.
    """
    try:
        result = await lib.build.BuildFn(build_context)
        return {"status": "success", "output": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
