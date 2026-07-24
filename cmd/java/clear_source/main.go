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
from pydantic import BaseModel, BaseSettings
import asyncio
import os
import shutil

app = FastAPI()

class Settings(BaseSettings):
    class Config:
        env_prefix = "buildpack_"

@app.get("/detect")
async def detect():
    # Implement detection logic similar to lib.DetectFn
    try:
        # Example detection criteria (to be updated based on actual requirements)
        return {"detected": True, "message": "Source clearing buildpack detected"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def build():
    try:
        # Example build logic similar to lib.BuildFn
        # Clear source code (non-blocking async implementation)
        loop = asyncio.get_event_loop()

        # Delete target directory
        await loop.run_in_executor(None, shutil.rmtree, "target", ignore_errors=True)

        # Delete temp directory
        await loop.run_in_executor(None, shutil.rmtree, "temp", ignore_errors=True)

        return {"status": "success", "message": "Source cleared successfully"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
