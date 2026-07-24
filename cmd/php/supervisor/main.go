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
Implements php/supervisor buildpack.
The supervisor buildpack installs the config needed for PHP runtime with supervisor.
"""

import logging
import os
import sys

from googlecloudplatform.buildpacks import main, detect_fn, build_fn

def main():
    """Main entry point for the PHP Supervisor buildpack."""
    try:
        # Set up basic logging configuration
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )

        # Parse command line arguments
        args = sys.argv[1:]

        # Run detection phase
        if detect_fn(args):
            print("Buildpack detected the application.")

            # Run build phase
            result = build_fn(args)
            if result:
                print("Build completed successfully.")
                return 0
            else:
                print("Error during build process.")
                return 1
        else:
            print("Application not detected by buildpack.")
            return 1
    except Exception as e:
        print(f"An error occurred: {str(e)}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
