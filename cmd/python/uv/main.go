# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Implements python/uv buildpack.
The uv buildpack installs dependencies using uv.
"""

import asyncio
from pathlib import Path

import lib  # Assuming this is relative import from same directory structure
from gcpbuildpack import gcp_main


async def main(detect_fn, build_fn):
    await gcp_main(detect_fn, build_fn)


if __name__ == "__main__":
    asyncio.run(main(lib.detect, lib.build))
