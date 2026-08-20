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

from fastapi import FastAPI, APIRouter, HTTPException
from pydantic import BaseModel
import asyncio
import lib

app = FastAPI()
router = APIRouter()

class BuildpackRequest(BaseModel):
    project_id: str
    source_dir: str = "/workspace"

@router.post("/v1/detect")
async def detect(request: BuildpackRequest):
    try:
        result = await asyncio.run_in_executor(None, lib.detect)
        return {"result": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/v1/build")
async def build(request: BuildpackRequest):
    try:
        result = await asyncio.run_in_executor(None, lib.build, request.source_dir)
        return {"result": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

app.include_router(router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
