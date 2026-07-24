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

from google.cloud import functions_framework
import os
import asyncio

@functions_framework.detect_and_build
async def main():
    from fastapi import FastAPI

    app = FastAPI(title="Functions Framework C++ Buildpack")

    if not os.getenv("FUNCTIONS_EMULATOR"):
        await asyncio.to_thread(uvicorn.run, app, host="0.0.0.0", port=8080)

if __name__ == "__main__":
    asyncio.run(main())
