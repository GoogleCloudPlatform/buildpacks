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

# Implements go/runtime buildpack.
# The runtime buildpack installs the Go toolchain.

import os
from pathlib import Path

import gcpbuildpack as gcp
from gcpbuildpack import runtime


def detect_fn(ctx: gcp.Context) -> gcp.DetectResult:
    """
    Detects if .go files are present or if runtime is set to go.
    
    Args:
        ctx: The context containing build information and environment variables.
        
    Returns:
        A DetectResult indicating whether the buildpack should be applied.
    """
    override_result = runtime.check_override("go")
    if override_result:
        return override_result
    
    source_dir = Path(ctx.working_directory)
    go_files = list(source_dir.glob("**/*.go"))
    
    if go_files:
        return gcp.OptIn("found .go files")
    
    return gcp.OptOut("no .go files found")


def build_fn(ctx: gcp.Context) -> None:
    """
    Installs the Go toolchain based on specified version.
    
    Args:
        ctx: The context containing build information and environment variables.
    """
    go_version = _get_go_version()
    layer_name = "go"
    
    # Create or get the existing layer
    layer = ctx.layers.get_layer(layer_name)
    
    # Install Go if not already cached
    runtime.install_tarball_if_not_cached(
        ctx,
        runtime.GO,
        go_version,
        layer
    )


def _get_go_version() -> str:
    """
    Determines the appropriate Go version to use.
    
    Returns:
        The Go version string.
        
    Raises:
        ValueError: If no valid version can be determined.
    """
    env_vars = {
        "GOOGLE_GO_VERSION": os.getenv("GOOGLE_GO_VERSION"),
        "GOOGLE_RUNTIME_VERSION": os.getenv("GOOGLE_RUNTIME_VERSION")
    }
    
    for var, value in env_vars.items():
        if value:
            return value
        
    # Default version if no environment variables are set
    return "1.21"


if __name__ == "__main__":
    pass  # This is a library module and should not be run directly
