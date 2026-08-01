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

"""Implements dart/compile buildpack."""

import os
import subprocess
from typing import Optional

import gcpbuildpack as gcp  # type: ignore
import dart_pkg as dart  # Assuming this is a Python package equivalent of the Go dart package
import env_pkg as env  # Assuming this is a Python package equivalent of the Go env package


def detect_fn(ctx: gcp.Context) -> tuple[Optional[gcp.DetectResult], Optional[str]]:
    """Detect function for checking if the buildpack should be used."""
    has_files, err = ctx.has_at_least_one("*.dart")
    if err:
        return None, f"Error finding .dart files: {err}"
    
    if not has_files:
        return gcp.OptOut("No .dart files found"), None
    
    return gcp.OptIn("Found .dart files"), None


def maybe_run_build_runner(ctx: gcp.Context, directory: str) -> None:
    """Runs build runner if applicable."""
    has_br, err = dart.has_build_runner(directory)
    if err:
        raise err
    if has_br:
        ctx.exec(["dart", "run", "build_runner", "build", "--delete-conflicting-outputs"], 
                 gcp.WithUserAttribution(), 
                 gcp.WithWorkDir(directory))


def build_fn(ctx: gcp.Context) -> None:
    """Build function for compiling the Dart application."""
    is_flutter, _ = dart.is_flutter(ctx.application_root())
    
    static_dir = ""
    server_dir = ""
    
    root_pubspec, err = dart.get_pubspec(ctx.application_root())
    if not err:
        has_buildpack = True
        if root_pubspec.buildpack and root_pubspec.buildpack.prebuild:
            ctx.exec(["sh", "-c", root_pubspec.buildpack.prebuild],
                     gcp.WithUserAttribution(),
                     gcp.WithWorkDir(ctx.application_root()))
        
        static_dir = os.path.join(ctx.application_root(), root_pubspec.buildpack.static)
        server_dir = os.path.join(ctx.application_root(), root_pubspec.buildpack.server)
        
        maybe_run_build_runner(ctx, static_dir)
        maybe_run_build_runner(ctx, server_dir)
    else:
        has_buildpack = False
        maybe_run_build_runner(ctx, ctx.application_root())
    
    # Create a layer for the compiled binary
    bl = ctx.create_layer("bin", gcp.LaunchLayer())
    if not bl:
        raise RuntimeError("Failed to create bin layer")
    
    bl.launch_environment.prepend("PATH", os.pathsep, bl.path)
    out_bin = os.path.join(bl.path, "server")
    
    # Determine buildable target
    buildable = os.environ.get(env.BUILDABLE, "bin/server.dart")
    
    # Build the server
    ctx.exec(["dart", "compile", "exe", buildable, "-o", out_bin],
             gcp.WithUserAttribution(),
             gcp.WithWorkDir(server_dir))
    
    ctx.add_web_process(["/bin/bash", "-c", out_bin])
    
    # Build Flutter web app if applicable
    if is_flutter and has_buildpack:
        ctx.exec(["flutter", "build", "web"],
                 gcp.WithUserAttribution(),
                 gcp.WithWorkDir(static_dir))
    
    # Run post-build script if available
    if has_buildpack and root_pubspec.buildpack.postbuild:
        ctx.exec(["sh", "-c", root_pubspec.buildpack.postbuild],
                 gcp.WithUserAttribution(),
                 gcp.WithWorkDir(bl.path))
