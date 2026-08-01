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
Implements go/build buildpack.
The build buildpack runs go build.
"""

import os
import subprocess
from pathlib import Path

import devmode
import env  # Assuming env module contains environment constants
import gcpbuildpack as gcp
import golang

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception | None]:
    """
    Detects if the buildpack should be used based on presence of .go files.
    Returns (gcp.OptIn or gcp.OptOut, error)
    """
    has_go_files = ctx.has_at_least_one("*.go")
    if not has_go_files:
        return gcp.opt_out("no .go files found"), None
    return gcp.opt_in("found .go files"), None

def build_fn(ctx: gcp.Context) -> Exception | None:
    """
    Builds the Go application.
    """
    try:
        # Keep GOCACHE in Devmode for faster rebuilds
        gocache_layer = ctx.layer("gocache", gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
        if devmode.enabled(ctx):
            gocache_layer.launch_environment.override("GOCACHE", gocache_layer.path)

        # Create a layer for the compiled binary
        bin_layer = ctx.layer("bin", gcp.LaunchLayer)

        buildable, err = go_buildable(ctx)
        if err:
            return Exception(f"unable to find a valid buildable: {err}")

        bld_flags = go_build_flags()
        workdir = os.getenv(golang.BuildDirEnv) or ctx.application_root()

        out_bin, bld, err = golang.perform_build(
            ctx,
            bin_layer,
            buildable,
            workdir,
            gocache_layer.path,
            bld_flags
        )
        if err:
            return err

        # Configure entrypoint for production
        if not devmode.enabled(ctx):
            ctx.add_web_process([out_bin])
            return None

        # Configure entrypoint and metadata for dev mode
        devmode.add_file_watcher_process(
            ctx,
            devmode.Config(
                build_cmd=bld,
                run_cmd=[out_bin],
                ext=devmode.GoWatchedExtensions
            )
        )

        return None
    except Exception as e:
        return e

def go_buildable(ctx: gcp.Context) -> tuple[str, Exception | None]:
    """
    Determines what to build based on environment and files.
    Returns (buildable path, error)
    """
    buildable = os.getenv(env.Buildable)
    if buildable is not None:
        return buildable, None

    try:
        buildables, err = search_buildables(ctx)
        if err:
            return "", err

        if len(buildables) == 1:
            return buildables[0], None
        elif not buildables:
            return ".", None
        else:
            # Multiple buildables - default to current directory
            return ".", None
    except Exception as e:
        return "", e

def search_buildables(ctx: gcp.Context) -> tuple[list[str], Exception | None]:
    """
    Searches for buildable Go packages.
    Returns (list of buildables, error)
    """
    try:
        result = ctx.exec(["go", "list", "-f", "{{if eq .Name \"main\"}}{{.Dir}}{{end}}", "./..."], gcp.WithUserAttribution)
        if result.error:
            return [], result.error

        app_root = Path(ctx.application_root()).resolve()
        buildables = []

        for dir in result.stdout.split():
            canonical_dir = Path(dir).resolve()
            rel_path = canonical_dir.relative_to(app_root)

            if str(rel_path) == ".":
                buildables.append(".")
            elif not str(rel_path).startswith(".."):
                buildables.append(f"./{rel_path}")
            else:
                buildables.append(str(rel_path))

        return buildables, None
    except Exception as e:
        return [], e

def go_build_flags() -> list[str]:
    """
    Returns the Go build flags from environment variables.
    """
    flags = []
    gcflags = os.getenv(env.GoGCFlags)
    if gcflags:
        flags.extend(["-gcflags", gcflags])
    ldflags = os.getenv(env.GoLDFlags)
    if ldflags:
        flags.extend(["-ldflags", ldflags])
    return flags
