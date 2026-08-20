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
import json
import logging
import asyncio
from typing import Dict, Any

from cmd.dotnet.appengine.lib import detect_fn, build_fn

app = FastAPI()

@app.post("/detect")
async def detect(request: Dict[str, Any]) -> Dict[str, Any]:
    """
    Detect function endpoint that identifies the build requirements.

    Args:
        request (Dict): Detection request containing context and metadata

    Returns:
        Dict: Detection results with platform info
    """
    try:
        logging.info("Starting detection process")

        # Simulate async file/network operations
        await asyncio.sleep(0.1)  # Replace with actual async operations

        result = detect_fn(request)
        logging.info("Detection completed successfully")
        return {"result": result}

    except Exception as e:
        logging.error(f"Detection failed: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def build(request: Dict[str, Any]) -> Dict[str, Any]:
    """
    Build function endpoint that executes the build process.

    Args:
        request (Dict): Build request containing context and metadata

    Returns:
        Dict: Build results with output files
    """
    try:
        logging.info("Starting build process")

        # Simulate async file/network operations
        await asyncio.sleep(0.1)  # Replace with actual async operations

        result = build_fn(request)
        logging.info("Build completed successfully")
        return {"result": result}

    except Exception as e:
        logging.error(f"Build failed: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/")
async def root() -> Dict[str, str]:
    """Root endpoint providing service info"""
    return {
        "service": "dotnet-appengine",
        "version": "1.0.0"
    }

if __name__ == "__main__":
    import uvicorn
    logging.info("Starting FastAPI server")
    uvicorn.run(app, host="0.0.0.0", port=8000)
