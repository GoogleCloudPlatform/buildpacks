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
Implements PHP runtime buildpack.
The runtime buildpack installs the PHP runtime.
"""

import argparse
import asyncio
from typing import Any

import lib


async def main() -> None:
    parser = argparse.ArgumentParser(description="PHP Runtime Buildpack")
    subparsers = parser.add_subparsers(dest="command", required=True)

    # Detect command
    detect_parser = subparsers.add_parser("detect", help="Detect PHP runtime requirements")
    detect_parser.set_defaults(func=lib.detect_fn)

    # Build command
    build_parser = subparsers.add_parser("build", help="Install PHP runtime")
    build_parser.set_defaults(func=lib.build_fn)

    args = parser.parse_args()

    if asyncio.iscoroutinefunction(args.func):
        await args.func()
    else:
        args.func()


if __name__ == "__main__":
    asyncio.run(main())
