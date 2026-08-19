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
Implements cpp/clear_source buildpack.
The clear_source buildpack deletes source files after building the application.
"""

import sys
from pathlib import Path

from googlecloudplatform.buildpacks import gcpbuildpack
from . import lib  # Assuming the same directory structure as Go

class Main(gcpbuildpack.Main):
    def __init__(self):
        super().__init__()

    def detect(self) -> bool:
        """
        Detect if this buildpack should run.
        Returns True if applicable, False otherwise.
        """
        return lib.detect()

    def build(self) -> None:
        """
        Execute the build process to clear source files.
        """
        lib.build()

def main() -> None:
    """
    Main function that initializes and runs the buildpack.
    """
    main_instance = Main()
    gcpbuildpack.run(main_instance)

if __name__ == "__main__":
    main()
