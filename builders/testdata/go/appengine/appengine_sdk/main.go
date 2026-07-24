# Copyright 2020 Google LLC
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

from fastapi import FastAPI, Request
import os
import asyncio

app = FastAPI()

class AppEngineResponse(BaseModel):
    message: str

@app.get("/")
async def handler(request: Request) -> AppEngineResponse:
    if 'GAE_APPLICATION' in os.environ:
        return AppEngineResponse(message="PASS")
    else:
        return AppEngineResponse(message="FAIL")

async def main():
    port = int(os.getenv("PORT", 8000))
    config = {
        "host": "0.0.0.0",
        "port": port,
        "reload": False
    }
    await asyncio.create_task(uvicorn.run(app, **config))

if __name__ == "__main__":
    asyncio.run(main())
