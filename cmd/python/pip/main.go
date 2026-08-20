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

from fastapi import FastAPI, APIRouter
from pydantic import BaseModel
import asyncio
import subprocess
from pathlib import Path

app = FastAPI()
router = APIRouter()

class BuildContext(BaseModel):
    project_path: str
    environment: dict

class EnvironmentConfig(BaseModel):
    pip_version: str
    requirements_file: str = "requirements.txt"

@router.get("/detect")
async def detect(context: BuildContext) -> bool:
    """Detect if pip buildpack is needed."""
    requirements_path = Path(context.project_path) / "requirements.txt"
    return await asyncio.to_thread(Path.exists, requirements_path)

@router.get("/build")
async def build(context: BuildContext, environment: EnvironmentConfig) -> dict:
    """Install dependencies using pip."""
    project_dir = Path(context.project_path)

    # Install pip dependencies
    process = await asyncio.create_subprocess_exec(
        "python",
        "-m",
        "pip",
        "install",
        "-r",
        str(project_dir / environment.requirements_file),
        "--user",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
        creationflags=asyncio.windows_utils.CREATE_NEW_PROCESS_GROUP if asyncio.get_event_loop().is_windows() else 0
    )

    await process.communicate()

    return {"status": "Dependencies installed successfully"}

app.include_router(router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
