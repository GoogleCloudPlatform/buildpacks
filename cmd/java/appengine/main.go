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

from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel, Field, validator
import json
import os
import asyncio

app = FastAPI()

class AppEngine:
    """
    Implements Java/App Engine buildpack functionality.
    Handles detection and building of App Engine applications.
    """

    class DetectRequest(BaseModel):
        """
        Request model for detection endpoint.
        """
        project_name: str = Field(..., description="Name of the GCP project")
        appengine_dir: str = Field(..., description="Path to App Engine configuration directory")

        @validator('appengine_dir')
        def validate_appengine_dir(cls, value):
            """Validate that App Engine directory exists."""
            if not os.path.exists(value):
                raise ValueError(f"App Engine directory {value} does not exist")
            return value

    class BuildRequest(BaseModel):
        """
        Request model for build endpoint.
        """
        entrypoint: str = Field(..., description="Entrypoint command for the application")

    async def detect(self, request: DetectRequest) -> dict:
        """
        Detect if the current directory is an App Engine project.

        Args:
            request (DetectRequest): Detection parameters

        Returns:
            dict: Detection result
        """
        try:
            # Check if it's an App Engine project
            is_appengine = os.path.exists(request.appengine_dir)

            return {
                "detected": is_appengine,
                "message": f"App Engine detected in {request.appengine_dir}" if is_appengine else "Not an App Engine project"
            }
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=str(e)
            )

    async def build(self, request: BuildRequest) -> dict:
        """
        Build the App Engine application.

        Args:
            request (BuildRequest): Build parameters

        Returns:
            dict: Build result
        """
        try:
            # Set the entrypoint in buildpack metadata
            metadata = {
                "entrypoint": request.entrypoint,
                "build_timestamp": asyncio.get_event_loop().time()
            }

            # Write metadata to file
            await self.write_metadata(metadata)

            return {
                "status": "success",
                "message": f"App Engine application built with entrypoint {request.entrypoint}"
            }
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=str(e)
            )

    async def write_metadata(self, metadata: dict) -> None:
        """
        Write build metadata to file asynchronously.

        Args:
            metadata (dict): Metadata to save
        """
        # Implement actual file writing logic here
        metadata_file = "buildpack.metadata.json"

        # Use asyncio for non-blocking I/O operations
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, lambda: json.dump(metadata, open(metadata_file, 'w')))

# Create FastAPI router for App Engine buildpack endpoints
appengine_router = FastAPI()

@appengine_router.post("/detect")
async def detect(request: AppEngine.DetectRequest):
    """Endpoint for detecting App Engine projects."""
    app_engine = AppEngine()
    return await app_engine.detect(request)

@appengine_router.post("/build")
async def build(request: AppEngine.BuildRequest):
    """Endpoint for building App Engine applications."""
    app_engine = AppEngine()
    return await app_engine.build(request)

# Include the App Engine router in the main application
app.include_router(appengine_router, prefix="/appengine", tags=["appengine"])

if __name__ == "__main__":
    import uvicorn

    # Run the FastAPI server with asyncio support
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        reload=True,
        workers=1
    )
