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
Implements PHP functions framework buildpack.
The functions_framework buildpack converts a function into an application and sets up the execution environment.
"""

import asyncio
import logging
import sys
from typing import Any, Dict

import aiofiles  # type: ignore
import subprocess  # nosec

# Simulate the detection and building process for PHP functions framework
async def detect() -> Dict[str, str]:
    """
    Detect if this buildpack applies to the current build.

    Returns:
        dict: Detection result with status and reason
    """
    try:
        # Simulated detection logic
        files = await aiofiles.os.listdir()
        has_composer = any(f == 'composer.json' for f in files)
        has_function = any(f == 'functions.php' for f in files)

        if not (has_composer or has_function):
            raise ValueError("No PHP function files detected")

        return {"status": "detected",
                "reason": "PHP functions framework buildpack applies"}

    except Exception as e:
        logging.error(f"Detection error: {str(e)}")
        sys.exit(1)

async def build() -> Dict[str, str]:
    """
    Build the PHP function application.

    Returns:
        dict: Build result with status and message
    """
    try:
        # Simulated build process
        # This would typically involve installing dependencies and setting up runtime
        logging.info("Starting build process")

        # Example of async file operations
        async with aiofiles.open('functions.php', 'r') as f:
            content = await f.read()
            if not content.strip():
                raise ValueError("Empty function file detected")

        # Simulate installing dependencies asynchronously
        proc = await asyncio.create_subprocess_exec(
            'composer', 'install',
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE)

        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise RuntimeError(f"Composer install failed: {stderr.decode()}")

        logging.info("Build completed successfully")
        return {"status": "success",
                "message": "PHP functions application built successfully"}

    except Exception as e:
        logging.error(f"Build error: {str(e)}")
        sys.exit(1)

async def main():
    """
    Main entry point for the buildpack.
    """
    # Run detection
    detection_result = await detect()

    if detection_result['status'] != 'detected':
        print("Skipping buildpack as it does not apply")
        return

    # Proceed with building
    build_result = await build()
    print(f"Build result: {build_result}")

if __name__ == "__main__":
    asyncio.run(main())
