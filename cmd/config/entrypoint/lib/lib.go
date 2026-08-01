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
Package lib implements config/entrypoint buildpack.
The entrypoint buildpack sets the image entrypoint based on environment variables or Procfile.
"""

import os
from typing import Dict, List, Optional

from gcpbuildpack import Context, DetectResult, OptIn, OptOut, UserError
from .appengine import Build as appengine_build
from .appyaml import get_entrypoint_from_file
from ..env import (
    GOOGLE_ENTRYPOINT,
    GAE_APPLICATION_YAML_PATH,
    XGOOGLE_TARGET_PLATFORM,
)

def detect(ctx: Context) -> DetectResult:
    """
    Determines if the buildpack should be applied based on environment and files.
    Returns OptIn or OptOut result.
    """
    # Detection for GAE and GCF
    is_gae = os.getenv("GAE_ENV") == "standard"
    is_gcf = os.getenv("FUNCTIONS_WORKER_POOL") is not None

    if is_gae or is_gcf:
        return OptIn({XGOOGLE_TARGET_PLATFORM: os.getenv(XGOOGLE_TARGET_PLATFORM)})

    # Check for GOOGLE_ENTRYPOINT
    entrypoint_env = os.getenv(GOOGLE_ENTRYPOINT)
    if entrypoint_env:
        return OptIn({GOOGLE_ENTRYPOINT: entrypoint_env})

    # Check for Procfile
    if ctx.file_exists("Procfile"):
        return OptIn({"Procfile": "found"})

    # Check app.yaml for entrypoint
    app_yaml_path = os.getenv(GOOGLE_APPLICATION_YAML_PATH, "app.yaml")
    if ctx.file_exists(app_yaml_path):
        entrypoint = get_entrypoint_from_file(ctx.application_root)
        if entrypoint:
            return OptIn({"app.yaml": f"entrypoint: {entrypoint}"})

    # If none of the above, opt out
    return OptOut(f"{GOOGLE_ENTRYPOINT} not set and no valid entrypoint found")

def build(ctx: Context) -> None:
    """
    Builds the application with the detected entrypoint.
    Modifies the context to add processes as needed.
    """
    is_gcf = os.getenv("FUNCTIONS_WORKER_POOL") is not None
    if is_gcf:
        return  # No action needed for GCF

    is_gae = os.getenv("GAE_ENV") == "standard"
    if is_gae:
        runtime = os.getenv("RUNTIME")
        if not runtime:
            raise UserError(f"Environment variable {XGOOGLE_TARGET_PLATFORM} is required for GAE.")
        appengine_build(ctx, runtime)
        return

    # Check environment variables
    entrypoint_env = os.getenv(GOOGLE_ENTRYPOINT)
    if entrypoint_env:
        ctx.add_process("web", [entrypoint_env], default=True)
        ctx.log(f"Using entrypoint from {GOOGLE_ENTRYPOINT}: {entrypoint_env}")
        return

    # Check Procfile
    if ctx.file_exists("Procfile"):
        content = ctx.read_file("Procfile")
        add_procfile_processes(ctx, content.decode())
        return

    # Check app.yaml for entrypoint
    app_yaml_path = os.getenv(GOOGLE_APPLICATION_YAML_PATH, "app.yaml")
    if ctx.file_exists(app_yaml_path):
        entrypoint = get_entrypoint_from_file(ctx.application_root)
        if entrypoint:
            ctx.add_process("web", [entrypoint], default=True)
            ctx.log("Using entrypoint from app.yaml.")
            return

    raise UserError(f"{GOOGLE_ENTRYPOINT} not set and no valid entrypoint found")

def add_procfile_processes(ctx: Context, content: str) -> None:
    """
    Parses Procfile content and adds processes to the context.
    Raises UserError if no web process is found.
    """
    lines = [line.strip() for line in content.splitlines()]
    processes: Dict[str, str] = {}

    for line in lines:
        # Skip empty lines and comments
        if not line or line.startswith('#'):
            continue

        parts = line.split(':', 1)
        if len(parts) != 2:
            continue  # Invalid format, skip

        name, cmd = parts[0].strip(), parts[1].strip()
        if not name or not cmd:
            continue

        processes[name] = cmd

    # Add all processes
    for name, cmd in processes.items():
        ctx.add_process(name, ["bash", "-c", cmd], default=(name == "web"))

    # Validate web process exists
    if "web" not in processes:
        raise UserError("No 'web' process found in Procfile")
