"""
Copyright 2025 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import os
import asyncio
from pathlib import Path
from pydantic import BaseModel

class AppConfig(BaseModel):
    java_home: str = "/usr/lib/jvm/java-11-openjdk-amd64"
    classpath: str = "target/classes"
    main_class: str = "com.example.Main"

async def detect(exploded_jar_path: Path) -> bool:
    """
    Detect if the current directory contains an exploded JAR application.
    """
    # Check for common Java exploded JAR patterns
    has_classes = any("classes" in f.name for f in exploded_jar_path.iterdir())
    has_resources = any("resources" in f.name for f in exploded_jar_path.iterdir())

    return has_classes and has_resources

async def build(exploded_jar_path: Path, output_dir: Path) -> None:
    """
    Build the exploded JAR application.
    """
    # Copy exploded JAR contents to output directory
    os.makedirs(output_dir, exist_ok=True)

    for item in exploded_jar_path.iterdir():
        if item.is_file():
            asyncio.create_task(copy_file(item, output_dir))
        else:
            asyncio.create_task(copy_directory(item, output_dir))

async def copy_file(src: Path, dest_dir: Path) -> None:
    """Asynchronously copy a file to the destination directory."""
    with open(src, 'rb') as f_in:
        content = await asyncio.to_thread(f_in.read)

    dest_path = dest_dir / src.name
    with open(dest_path, 'wb') as f_out:
        await asyncio.to_thread(f_out.write, content)

async def copy_directory(src: Path, dest_dir: Path) -> None:
    """Asynchronously copy a directory to the destination directory."""
    dest_path = dest_dir / src.name
    os.makedirs(dest_path, exist_ok=True)

    for item in src.iterdir():
        if item.is_file():
            asyncio.create_task(copy_file(item, dest_path))
        else:
            asyncio.create_task(copy_directory(item, dest_path))

class ExplodedJarBuildpack:
    async def detect(self) -> bool:
        return await detect(Path("."))

    async def build(self, output_dir: Path) -> None:
        return await build(Path("."), output_dir)

async def main():
    config = AppConfig()
    buildpack = ExplodedJarBuildpack()

    if await buildpack.detect():
        print("Detected exploded JAR application")
        await buildpack.build(Path("output"))
    else:
        print("No exploded JAR application found")

if __name__ == "__main__":
    asyncio.run(main())
