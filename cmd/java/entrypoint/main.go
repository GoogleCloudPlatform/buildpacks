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

"""Implements the java/entrypoint buildpack."""

import os
from google.cloud.buildpacks import appengine
from google.cloud.buildpacks import appyaml
from google.cloud.buildpacks import devmode
from google.cloud.buildpacks import env
from google.cloud.buildpacks import gcpbuildpack as gcp
from google.cloud.buildpacks import java

def DetectFn(ctx: gcp.Context) -> (gcp.DetectResult, Exception):
    """Exported detect function."""
    return gcp.OptInAlways(), None

def BuildFn(ctx: gcp.Context) -> Exception:
    """Exported build function."""
    entrypoint = _getEntrypoint(ctx)
    
    if env.IsFlex() and entrypoint:
        ctx.Setenv(env.Entrypoint, entrypoint)
        appengine.Build(ctx, "java", None)
        return None

    try:
        executable = java.ExecutableJar(ctx)
    except Exception as e:
        raise ValueError(f"Finding executable jar: {e}") from e

    command = ["java", "-jar", executable]

    if devmode.Enabled(ctx):
        config = devmode.Config(
            BuildCmd=[".devmode_rebuild.sh"],
            RunCmd=command,
            Ext=devmode.JavaWatchedExtensions
        )
        try:
            devmode.AddFileWatcherProcess(ctx, config)
        except Exception as e:
            raise ValueError(f"Adding devmode file watcher: {e}") from e
        return None

    ctx.AddWebProcess(command)
    return None

def _getEntrypoint(ctx: gcp.Context) -> str:
    """Gets the entrypoint from environment or app.yaml."""
    entrypoint = os.getenv(env.Entrypoint, "")
    if not entrypoint:
        entrypoint, _ = appyaml.EntrypointIfExists(ctx.ApplicationRoot())
    return entrypoint
