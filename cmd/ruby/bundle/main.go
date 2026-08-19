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

"""Implements ruby/bundle buildpack.
The bundle buildpack installs dependencies using bundle.
"""

import gcp_buildpack as gcp  # Assuming GCP buildpack is converted to Python

from .lib import detect, build  # Importing functions from lib package


def main():
    """Main entry point for the buildpack."""
    gcp.main(detect_fn=detect, build_fn=build)


if __name__ == "__main__":
    main()
