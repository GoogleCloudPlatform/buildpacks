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
Implements Python runtime buildpack.
The runtime buildpack installs the Python runtime.
"""

from googlecloudplatform.buildpacks.core.decorators import gcp_main
from .lib import detect, build

@gcp_main(detect.DetectFn, build.BuildFn)
def main():
    """
    Main function for the Python runtime buildpack.
    Handles detection and installation of the Python runtime.
    """
    pass  # Detection and building handled by decorators
