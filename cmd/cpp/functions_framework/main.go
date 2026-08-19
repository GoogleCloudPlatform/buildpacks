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
Implements cpp/functions_framework buildpack.
The functions_framework buildpack converts a function into an application and sets up the execution environment.
"""

import lib  # Assuming this is properly imported from the equivalent Python package
from gcpbuildpack import BaseBuilder

class FunctionsFrameworkBuildpack(BaseBuilder):
    def detect(self) -> bool:
        """
        Detects if the current environment requires this buildpack.
        Returns True if applicable, False otherwise.
        """
        return lib.detect_fn()

    async def build(self) -> None:
        """
        Builds the application using the functions framework.
        This method is async and should be awaited where used.
        """
        await lib.build_fn()

if __name__ == "__main__":
    FunctionsFrameworkBuildpack().run()
