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

"""
Implements Ruby App Engine buildpack.
The App Engine buildpack sets the image entrypoint.
"""

import sys
import asyncio
from typing import Any

from fastapi import FastAPI
from pydantic import BaseModel, BaseSettings
import aiofiles  # For async file operations
import httpx     # For async HTTP requests

class Settings(BaseSettings):
    """
    Configuration settings for the Ruby App Engine buildpack.
    """
    project_id: str
    service_name: str = "appengine"

    class Config:
        env_prefix = "GOOGLE_"

settings = Settings()

async def detect_fn() -> dict[str, Any]:
    """
    Detect function that identifies if the current environment is suitable for Ruby App Engine.
    Returns a dictionary with detection results.
    """
    try:
        # Simulate detection logic
        async with aiofiles.open("requirements.txt", mode="r") as f:
            content = await f.read()
            return {"detected": "ruby" in content, "version": "3.11"}
    except FileNotFoundError:
        return {"detected": False}

async def build_fn() -> None:
    """
    Build function that sets up the Ruby environment for App Engine.
    """
    try:
        # Perform async file operations
        async with aiofiles.open("Dockerfile", mode="w") as f:
            await f.write("FROM ruby:3.11\n")

        # Perform async HTTP requests if needed
        async with httpx.AsyncClient() as client:
            response = await client.get("https://rubygems.org/api/v2/specs.json")
            print(f"RubyGems API response status: {response.status_code}")

    except Exception as e:
        print(f"Error during build phase: {str(e)}", file=sys.stderr)
        sys.exit(1)

async def main() -> None:
    """
    Main entrypoint for the Ruby App Engine buildpack.
    """
    try:
        # Run detection
        detected = await detect_fn()
        if not detected.get("detected"):
            print("Ruby environment not detected. Exiting.")
            return

        # Proceed with building
        print(f"Detected Ruby {detected['version']}. Building...")
        await build_fn()

    except Exception as e:
        print(f"Error in main: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())
