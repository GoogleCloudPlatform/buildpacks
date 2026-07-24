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
Implements the java/entrypoint buildpack.
"""

import sys

from . import lib
from gcpbuildpack import main as gcp_main


def main():
    """
    Main function that initializes and runs the buildpack process.

    This function sets up the environment, detects the application type,
    and builds the deployment package using the specified functions from
    the lib module.
    """
    try:
        # Call the Google Cloud Platform buildpack main function with
        # detection and build functions from the lib module.
        gcp_main(lib.detect_fn, lib.build_fn)
    except Exception as e:
        # Handle any exceptions that occur during the build process
        print(f"Error: {str(e)}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
