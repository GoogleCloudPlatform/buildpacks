"""
Implements clear_source buildpack functionality using FastAPI.

The clear_source buildpack deletes source files after building the application.

Copyright 2025 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at:

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
import logging

app = FastAPI()

# Simulating detection logic from lib.DetectFn
class DetectResponse(BaseModel):
    id: str = "clear_source"
    message: str = "Clearing source files"

@app.get("/detect")
async def detect():
    return DetectResponse()

# Simulating build logic from lib.BuildFn
class BuildResponse(BaseModel):
    status: str = "success"
    message: str = "Source files cleared successfully"

async def delete_files(file_paths: list[str]):
    """Asynchronously deletes multiple files."""
    tasks = []
    for file_path in file_paths:
        tasks.append(asyncio.create_task(delete_file(file_path)))
    await asyncio.gather(*tasks)

async def delete_file(file_path: str):
    """Deletes a single file asynchronously."""
    try:
        # Simulate file deletion
        print(f"Deleting {file_path}")
        await asyncio.sleep(0.1)  # Simulate I/O delay
    except Exception as e:
        logging.error(f"Failed to delete {file_path}: {str(e)}")
        raise HTTPException(
            status_code=400,
            detail=f"Error deleting file: {file_path}"
        )

@app.post("/build")
async def build(request_data: dict):
    try:
        # Simulate processing files
        file_paths = request_data.get("files", [])
        await delete_files(file_paths)
        return BuildResponse()
    except Exception as e:
        logging.error(f"Build failed: {str(e)}")
        raise HTTPException(
            status_code=400,
            detail=str(e)
        )

if __name__ == "__main__":
    import uvicorn
    print("Starting clear_source buildpack server on http://localhost:8080")
    uvicorn.run(app, host="0.0.0.0", port=8080)
