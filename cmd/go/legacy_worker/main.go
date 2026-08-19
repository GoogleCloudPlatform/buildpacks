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
Implements the legacy GCF Go 1.11 worker buildpack.
The legacy_worker buildpack converts a function into an application and sets up the execution environment.
"""

import logging
from typing import Dict, Any

import subprocess
import json
import os

logger = logging.getLogger(__name__)

def detect_fn() -> Dict[str, str]:
    """
    Detects if the current environment is suitable for GCF Go 1.11.
    Returns a dictionary of detection results or None if not applicable.
    """
    # Implement detection logic here
    # For example, check for GO_VERSION in environment variables
    go_version = os.environ.get('GO_VERSION', '')
    if not go_version.startswith('1.11'):
        logger.info("Not using GCF Go 1.11 worker")
        return None

    return {
        'builder': 'gcp/golang111-legacy-worker',
        'version': '1.0'
    }

def build_fn() -> None:
    """
    Builds the legacy worker environment.
    Sets up the necessary files and dependencies.
    """
    # Setup worker binary
    worker_binary = "worker"
    with open(worker_binary, 'w') as f:
        f.write("#!/bin/bash\n")
        f.write("echo 'Legacy GCF Go 1.11 worker'\n")
    os.chmod(worker_binary, 0o755)

    # Install dependencies
    subprocess.run(["pip", "install", "-r", "requirements.txt"], check=True)

def main() -> None:
    """
    Main entry point for the legacy worker buildpack.
    Handles detection and building process.
    """
    try:
        detection_result = detect_fn()
        if not detection_result:
            return

        logger.info("Detected GCF Go 1.11 environment")
        logger.info(f"Buildpack version: {detection_result['version']}")

        build_fn()

    except Exception as e:
        logger.error(f"Error during build process: {str(e)}")
        raise

if __name__ == "__main__":
    main()
