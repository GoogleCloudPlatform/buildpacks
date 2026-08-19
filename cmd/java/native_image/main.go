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
Implements Java GraalVM Native Image buildpack.
This buildpack installs the GraalVM compiler into a layer and builds a native image of the Java application.
"""

import googlecloudplatform.buildpacks.cmd.java.native_image.lib as lib
from googlecloudplatform.buildpacks.pkg import gcpbuildpack

def main():
    """
    Main entry point for the buildpack.
    Detects if the buildpack applies and performs the build process.
    """
    try:
        gcpbuildpack.detect_and_build(lib.DetectFn, lib.BuildFn)
    except Exception as e:
        print(f"Error during buildpack execution: {e}")
        raise

if __name__ == "__main__":
    main()
