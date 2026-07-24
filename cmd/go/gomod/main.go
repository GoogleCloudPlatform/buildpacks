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
from pathlib import Path

app = FastAPI()

class DetectRequest(BaseModel):
    source_path: str

class DetectResponse(BaseModel):
    is_go_module: bool

class BuildRequest(BaseModel):
    source_path: str

class BuildResponse(BaseModel):
    status: str
    error: str | None = None

@app.post("/detect")
async def detect(request: DetectRequest) -> DetectResponse:
    try:
        # Asynchronously check for go.mod file
        source_path = Path(request.source_path)
        go_mod_exists = await asyncio.get_event_loop().run_in_executor(
            None,
            lambda: (source_path / "go.mod").exists()
        )
        return DetectResponse(is_go_module=go_mod_exists)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def build(request: BuildRequest) -> BuildResponse:
    try:
        # Asynchronously download go modules
        source_path = Path(request.source_path)

        # Run go mod download asynchronously
        proc = await asyncio.create_subprocess_exec(
            "go", "mod", "download",
            cwd=source_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )

        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            return BuildResponse(status="error", error=stderr.decode())

        return BuildResponse(status="success")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
