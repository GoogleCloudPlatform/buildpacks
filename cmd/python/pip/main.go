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
Implements python/pip buildpack.
The pip buildpack installs dependencies using pip.
"""

import sys
from typing import Dict, Any

import gcp  # type: ignore
from cmd.python.pip.lib import detect_fn, build_fn


class PipBuildpack:
    def __init__(self) -> None:
        self._detect = detect_fn.DetectFn()
        self._build = build_fn.BuildFn()

    async def detect(self, context: Dict[str, Any]) -> Dict[str, Any]:
        return await self._detect(context)

    async def build(self, context: Dict[str, Any]) -> Dict[str, Any]:
        return await self._build(context)


if __name__ == "__main__":
    gcp.main(PipBuildpack())
