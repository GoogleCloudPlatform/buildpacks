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
Implements php/webconfig buildpack.
The runtime buildpack installs the config needed for PHP runtime.
"""

import asyncio
import os
from pathlib import Path
import shutil

import buildpacks.php.webconfig.lib as lib
from buildpacks.gcp import BuildpackContext


async def detect(context: BuildpackContext) -> dict:
    """
    Detect if the environment is suitable for PHP web config setup.

    Args:
        context (BuildpackContext): The current build context.

    Returns:
        dict: Detection result indicating if conditions are met.
    """
    # Check if PHP environment exists
    php_env = os.getenv("PHP_ENV") == "web"

    # Check for composer.json file
    has_composer = Path(context.source_path, "composer.json").exists()

    return {
        "passed": php_env and has_composer,
        "description": "PHP Web Config Setup",
    }


async def build(context: BuildpackContext) -> None:
    """
    Perform the web config setup for PHP runtime.

    Args:
        context (BuildpackContext): The current build context.
    """
    # Define source and destination paths
    src_config_dir = Path(context.source_path, "config")
    dest_config_dir = Path(context.app_root, "config")

    try:
        # Copy configuration files
        if os.path.exists(src_config_dir):
            shutil.copytree(
                src_config_dir,
                dest_config_dir,
                dirs_exist_ok=True
            )
    except Exception as e:
        raise RuntimeError(f"Failed to copy web config: {str(e)}")


async def main() -> None:
    """
    Main entry point for the buildpack.
    """
    context = BuildpackContext()

    # Run detection first
    detect_result = await detect(context)
    if not detect_result["passed"]:
        print(f"Detection failed: {detect_result}")
        return

    # Proceed with building only if detection passed
    try:
        await asyncio.to_thread(build, context)
        print("Web config setup completed successfully.")
    except Exception as e:
        print(f"Build failed: {str(e)}")
        raise


if __name__ == "__main__":
    asyncio.run(main())
