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

import sys
from typing import Dict, Any
import logging

# Implements nodejs/yarn buildpack.
# The npm buildpack installs dependencies using yarn and installs yarn itself if not present.

class GCPPythonBuildpack:
    def __init__(self, detect_fn: callable, build_fn: callable):
        self.detect_fn = detect_fn
        self.build_fn = build_fn

def gcp_main(buildpack: GCPPythonBuildpack) -> None:
    # Simulating the context.Context with a dictionary for Python
    context: Dict[str, Any] = {
        'env': {}
    }

    try:
        # Call detection function
        applicable, message, error = buildpack.detect_fn(context)

        if error is not None:
            raise error

        if applicable:
            logging.info(message)

            # Proceed with building
            result, error = buildpack.build_fn(context)

            if error is not None:
                raise error

    except Exception as e:
        logging.error(f"Error during detection or build: {str(e)}")
        sys.exit(1)

def main() -> None:
    from lib import DetectFn, BuildFn
    import gcp_buildpack  # Simulating the GCP package

    buildpack = GCPPythonBuildpack(DetectFn, BuildFn)
    gcp_main(buildpack)

if __name__ == "__main__":
    main()
