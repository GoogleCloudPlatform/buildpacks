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

"""Implements utils/archive-source buildpack. Archives user's source code."""

import os
import shutil
import subprocess

from cmd.utils.archive_source import gcp

ARCHIVE_NAME = "source-code.tar.gz"

def detect(context: gcp.Context) -> bool:
    """Detects if this buildpack should be applied."""
    if not context.is_gcf():
        return False
    
    clear_source = os.getenv(gcp.CLEAR_SOURCE)
    if clear_source is not None:
        try:
            clear = clear_source.lower() in {"true", "1", "yes"}
        except ValueError as err:
            raise gcp.UserError(f"Failed to parse {gcp.CLEAR_SOURCE}: {err}") from err
        if clear:
            return False
    
    return True

def build(context: gcp.Context) -> None:
    """Builds the archive layer."""
    try:
        layer = context.layer("src", gcp.LayerType.LAUNCH)
    except Exception as err:
        raise RuntimeError(f"Creating layer failed: {err}") from err
    
    file_path = os.path.join(layer.path, ARCHIVE_NAME)
    source_dir = context.application_root()
    
    if archive_source(context, file_path, source_dir):
        # Symlink the archive to /workspace/.googlebuild
        google_build_path = ".googlebuild"
        os.makedirs(google_build_path, exist_ok=True)
        
        stable_path = os.path.join(
            context.application_root(), google_build_path, ARCHIVE_NAME
        )
        try:
            if os.path.exists(stable_path):
                os.remove(stable_path)
            os.symlink(file_path, stable_path)
        except Exception as err:
            raise RuntimeError(f"Creating symlink failed: {err}") from err
        
        context.add_label("source-archive", stable_path)

def archive_source(context: gcp.Context, file_name: str, dir_name: str) -> bool:
    """Archives user's source code in a layer."""
    try:
        subprocess.run(
            [
                "tar",
                "--create",
                "--gzip",
                "--preserve-permissions",
                f"--file={file_name}",
                "--directory",
                dir_name,
                "."
            ],
            check=True
        )
        return True
    except subprocess.CalledError as err:
        raise RuntimeError(f"Archiving source failed: {err}") from err

def run_buildpack(detect_fn, build_fn):
    """Runs the buildpack with detection and build functions."""
    context = gcp.Context()
    if detect_fn(context):
        build_fn(context)
