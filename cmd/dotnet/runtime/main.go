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

import asyncio

from fastapi import FastAPI
from pydantic import BaseModel
import lib  # Assuming this is your Python library package
import gcpbuildpack  # Assuming this is your Python GCP buildpack package

async def main():
    await lib.detect_fn()
    await lib.build_fn()

if __name__ == "__main__":
    asyncio.run(main())
