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

"""Implements utils/nginx buildpack. Installs nginx web server and pid1 binaries."""

import os
import sys
from pathlib import Path

import gcpbuildpack as gcp
from libcnb import Layer
import env
import runtime
import static


DEFAULT_NGINX_VERSION_CONSTRAINT = "^1.21.6"
PID1_VERSION_CONSTRAINT = "^1.0.0"
DEFAULT_NGINX_INSTALLABLE_RUNTIME = runtime.Nginx
STATIC_NGINX_INSTALLABLE_RUNTIME = runtime.CanonicalNginx

def DetectFn(context: gcp.Context) -> bool:
    """Detect if the nginx buildpack should be used."""
    return True  # Always opt in

def BuildFn(context: gcp.Context) -> None:
    """Build function for installing nginx and related components."""
    using_static_serve, err = env.UsingStaticServe()
    if err is not None:
        context.Warn(f"Failed to parse GOOGLE_STATIC_SERVE: {err}, defaulting to false")
        using_static_serve = False

    nginx_version_constraint = DEFAULT_NGINX_VERSION_CONSTRAINT
    nginx_installable_runtime = DEFAULT_NGINX_INSTALLABLE_RUNTIME

    if using_static_serve:
        runtime_name = os.getenv(env.RUNTIME)
        nginx_version_constraint = static.GetNginxVersionConstraint(runtime_name)
        nginx_installable_runtime = STATIC_NGINX_INSTALLABLE_RUNTIME

    if env.IsStaticBaseImage():
        context.Log("Skipping nginx install for static base image.")
    else:
        context.Log(f"Installing nginx: {nginx_installable_runtime}")
        layer, err = Install(context, "nginx", nginx_version_constraint, nginx_installable_runtime)
        if err is not None:
            raise ValueError(f"Failed to install nginx: {err}")

        # Update launch environment
        layer.launch_environment.Append("PATH", os.pathsep, str(Path(layer.path) / "sbin"))
        layer.build_environment.Default("NGINX_ROOT", str(layer.path))

    if not using_static_serve:
        context.Log("Installing pid1")
        layer, err = Install(context, "pid1", PID1_VERSION_CONSTRAINT, runtime.Pid1)
        if err is not None:
            raise ValueError(f"Failed to install pid1: {err}")

        # Update launch environment
        layer.launch_environment.Append("PATH", os.pathsep, str(Path(layer.path)))
        layer.build_environment.Default("PID1_DIR", str(layer.path))

def Install(context: gcp.Context, name: str, version_constraint: str, installable_runtime: runtime.InstallableRuntime) -> tuple[Layer, None]:
    """Install software using the specified parameters."""
    try:
        # Create or get existing layer
        layer = context.Layer(name=name, layers=[gcp.BUILD_LAYER, gcp.CACHE_LAYER, gcp.LAUNCH_LAYER])
        
        # Install tarball if not cached
        runtime.InstallTarballIfNotCached(context, installable_runtime, version_constraint, layer)
        return layer, None
    except Exception as err:
        context.Log(f"Error installing {name}: {err}")
        return None, err

if __name__ == "__main__":
    # This is a library module and should not be run directly
    sys.exit(1)
