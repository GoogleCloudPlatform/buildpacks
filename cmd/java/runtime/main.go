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
Implements java/runtime buildpack.
The runtime buildpack installs the JDK.
"""

from googlecloudplatform.buildpacks.runtime.java.lib import detect_fn as detect
from googlecloudplatform.buildpacks.runtime.java.lib import build_fn as build
import gcpbuildpack

def main():
    """Main entry point for the Java runtime buildpack."""
    # Initialize and run the buildpack with detection and build functions.
    bp = gcpbuildpack.Buildpack()
    bp.initialize(detect, build)
    bp.run()

if __name__ == '__main__':
    main()
