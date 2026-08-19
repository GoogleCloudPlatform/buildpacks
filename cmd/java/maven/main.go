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

"""Implements java/maven buildpack. The Maven buildpack builds Maven applications."""

import asyncio
from .lib import detect_fn, build_fn
from gcpbuildpack.gcp import main as gcp_main

async def main():
    """Main entry point for the Maven buildpack."""
    await gcp_main(detect_fn=detect_fn, build_fn=build_fn)

def run():
    """Run the async main function."""
    asyncio.run(main())

if __name__ == "__main__":
    run()
