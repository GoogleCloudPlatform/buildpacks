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
Implements php/composer-install buildpack.
The composer-install buildpack installs the composer dependency manager.
"""

import sys
from .lib import detect_fn, build_fn
from ..gcpbuildpack import gcp_main

def main():
    try:
        # Run detection phase
        if not detect_fn():
            print("Skipping buildpack as no PHP project detected.")
            return

        # Proceed with the build
        build_fn()

    except Exception as e:
        print(f"Error during build: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
