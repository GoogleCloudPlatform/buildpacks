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

import os
import sys
import logging

from googlecloudplatform.buildpacks import cmd.python.functions_framework_compat.lib as lib

def main():
    try:
        # Call detect function first
        if not lib.DetectFn():
            return

        # Proceed with build
        lib.BuildFn()

    except Exception as e:
        logging.error(f"Error during execution: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()
