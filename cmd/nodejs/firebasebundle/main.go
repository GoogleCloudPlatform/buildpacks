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
Implements nodejs/firebasebundle buildpack.
The output bundle buildpack sets up the output bundle for future steps
It will do the following:
1. Copy over static assets to the output bundle dir
2. Override run script with a new one to run the optimized build
"""

import logging
from argparse import ArgumentParser

import googlecloudplatform.buildpacks.gcpbuildpack as gcp
from googlecloudplatform.buildpacks.nodejs.firebasebundle import lib

logger = logging.getLogger(__name__)

def detect_fn():
    """Detect function implementation"""
    return None  # No detection needed for this buildpack

def build_fn(output_dir, env):
    """Build function implementation"""
    try:
        # Copy static assets
        lib.copy_static_assets(output_dir)

        # Override run script
        lib.override_run_script(output_dir, env)

        logger.info("Firebase bundle setup completed successfully")
        return None

    except Exception as e:
        logger.error(f"Error setting up firebase bundle: {str(e)}")
        return str(e)

def main():
    """Main function"""
    parser = ArgumentParser(description='Firebase Bundle Buildpack')
    parser.add_argument('--verbose', action='store_true', help='Enable verbose logging')
    parser.add_argument('--version', action='store_true', help='Print version and exit')
    args = parser.parse_args()

    if args.version:
        print("FirebaseBundleBuildpack 1.0.0")
        return

    gcp.main(detect_fn, build_fn)

if __name__ == "__main__":
    main()
