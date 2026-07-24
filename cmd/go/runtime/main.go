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
Implements go/runtime buildpack.
The runtime buildpack installs the Go toolchain.
"""

import logging

from googlecloudplatform.buildpacks import gcpbuildpack
from googlecloudplatform.buildpacks.cmd.go.runtime import lib

def main():
    """
    Main function for the Go runtime buildpack.
    Detects and builds the Go application.
    """
    try:
        # Detect the Go environment
        detect_result = lib.detect_fn()
        logging.info("Go environment detected successfully: %s", detect_result)

        # Build the Go application
        build_result = lib.build_fn()
        logging.info("Go application built successfully: %s", build_result)

    except Exception as e:
        logging.error("Error in Go runtime buildpack: %s", str(e))
        raise

if __name__ == "__main__":
    main()
