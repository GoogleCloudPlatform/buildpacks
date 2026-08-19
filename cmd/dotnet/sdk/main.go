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
Binary dotnet/runtime buildpack detects .NET applications
and install the corresponding version of .NET runtime.
"""

import os
import logging

from . import detect  # type: ignore
from . import build   # type: ignore

def main():
    """Main entry point for the .NET buildpack."""
    print("Running dotnet buildpack")

    # Check if debug mode is enabled
    debug = os.getenv('GOOGLE_BUILDpackS_DEBUG', 'false').lower() == 'true'
    if debug:
        logging.basicConfig(level=logging.DEBUG)

    try:
        print("Detecting .NET application...")
        detection_result = detect.run()
        print(f"Detection completed: {detection_result}")

        # Proceed with build only if detection was successful
        if detection_result.get('success', False):
            print("Building .NET application...")
            build_result = build.run(detection_result)
            print(f"Build completed: {build_result}")

    except Exception as e:
        print(f"Error occurred: {str(e)}")
        raise

if __name__ == "__main__":
    main()
