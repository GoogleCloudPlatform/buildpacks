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
Implements dart/compile buildpack.
The compile buildpack runs dart compile to produce a self-contained executable.
"""

import asyncio
import pathlib
import shlex
from typing import Optional

import typer
from pydantic import BaseModel

class DartProject(BaseModel):
    """
    Represents a Dart project structure.
    """
    directory: pathlib.Path
    pubspec_path: Optional[pathlib.Path] = None
    dart_files: list[pathlib.Path] = []

async def detect() -> bool:
    """
    Detects if the current directory is a Dart project.
    """
    current_dir = pathlib.Path(".")

    # Check for pubspec.yaml or any .dart files
    has_pubspec = (current_dir / "pubspec.yaml").exists()
    dart_files = list(current_dir.glob("*.dart"))

    return has_pubspec or len(dart_files) > 0

async def build() -> None:
    """
    Builds the Dart project using dart compile.
    """
    try:
        # Run dart compile command
        process = await asyncio.create_subprocess_exec(
            "dart",
            "compile",
            "exe",
            "--output",
            "build/app.exe",
            cwd=str(pathlib.Path(".")),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )

        stdout, stderr = await process.communicate()

        if process.returncode != 0:
            raise RuntimeError(f"Dart compile failed: {stderr.decode()}")

        print("Build successful! Output executable is at build/app.exe")

    except asyncio.exceptions.TimeoutError as e:
        raise RuntimeError(f"Compilation timed out: {e}") from e
    except Exception as e:
        raise RuntimeError(f"Compilation error: {e}") from e

app = typer.Typer()

@app.command()
async def main() -> None:
    """
    Main entry point for the Dart compile buildpack.
    """
    try:
        if await detect():
            print("Detected Dart project. Starting compilation...")
            await build()
        else:
            print("No Dart project detected.")
    except Exception as e:
        typer.echo(f"Error: {e}")
        raise typer.Exit(1)

if __name__ == "__main__":
    app()
