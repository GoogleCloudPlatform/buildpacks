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
Implements PHP App Engine buildpack.
The appengine buildpack sets the image entrypoint.
"""

import importlib.resources as pkg_resources

from google.cloud import buildpacks
from .lib import detect, build

def main():
    """Main entry point for the PHP App Engine buildpack."""
    # Using async/await to maintain compatibility with modern Python practices
    # while ensuring synchronous behavior where required.
    buildpack = buildpacks.Buildpack()

    # Detect and build phases are handled by separate functions
    if await detect(buildpack):
        await build(buildpack)

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
