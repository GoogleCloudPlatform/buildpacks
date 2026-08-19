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
Implements ruby/bundle buildpack.
The bundle buildpack installs dependencies using bundle.
"""

import sys
from pathlib import Path

sys.path.append(str(Path(__file__).parent.parent.parent))  # Add project root directory to Python path

from cmd.ruby.rubygems.lib import detect, build
from gcpbuildpack import buildpack_main

def main():
    """
    Main function for the ruby bundle buildpack.

    This function initializes and runs the buildpack with the detection and build functions.
    """
    buildpack_main(detect, build)

if __name__ == "__main__":
    main()
