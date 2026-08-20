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

import asyncio
import os
from typing import Optional

import click
from pydantic import BaseModel, Field

class BuildpackConfig(BaseModel):
    entrypoint: str = Field(..., description="The command to run")
    procfile_path: str = Field("Procfile", description="Path to Procfile")
    environment_vars: dict[str, str] = Field(
        default_factory=dict,
        description="Environment variables for the buildpack"
    )

class EntryPointBuilder:
    def __init__(self, source_dir: str):
        self.source_dir = source_dir
        self.config = BuildpackConfig()

    async def detect(self) -> bool:
        # Check if Procfile exists or environment variables are set
        procfile_path = os.path.join(self.source_dir, self.config.procfile_path)
        return (
            os.path.exists(procfile_path) or
            len(os.environ.get("ENTRYPOINT", "")) > 0
        )

    async def build(self):
        # Set entrypoint based on Procfile or environment variables
        if "ENTRYPOINT" in os.environ:
            self.config.entrypoint = os.environ["ENTRYPOINT"]
        else:
            procfile_path = os.path.join(self.source_dir, self.config.procfile_path)
            with open(procfile_path, "r") as f:
                # Read first line of Procfile
                self.config.entrypoint = f.readline().strip()

async def main_async(source_dir: str):
    builder = EntryPointBuilder(source_dir)
    if await builder.detect():
        await builder.build()
        print(f"Set entrypoint to: {builder.config.entrypoint}")
    else:
        print("No entrypoint detected, buildpack skipped.")

@click.command()
@click.option(
    "--source",
    "-s",
    type=click.Path(exists=True),
    required=True,
    help="Path to source directory"
)
def main(source):
    try:
        asyncio.run(main_async(source))
    except Exception as e:
        raise click.ClickException(str(e))

if __name__ == "__main__":
    main()
