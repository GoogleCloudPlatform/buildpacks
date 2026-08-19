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
Implements php/runtime buildpack.
The runtime buildpack installs the PHP runtime.
"""

import os
from typing import Any

import gcp  # Assuming this is the appropriate Python package structure
from .lib import detect_fn, build_fn


class BuildpackRuntime:
    def __init__(self) -> None:
        self.name = "php"
        self.version = "7.4"

    async def detect(self) -> dict[str, Any]:
        """
        Detects if the PHP runtime is required based on the application's requirements.
        Returns a dictionary with detection results.
        """
        return await detect_fn()

    async def build(self, context: dict[str, Any]) -> None:
        """
        Installs the PHP runtime based on the detected requirements.
        """
        await build_fn(context)


def main() -> None:
    runtime = BuildpackRuntime()
    gcp.main(runtime.detect, runtime.build)


if __name__ == "__main__":
    main()
