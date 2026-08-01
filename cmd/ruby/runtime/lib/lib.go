# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Implements ruby/runtime buildpack. The runtime buildpack installs the Ruby runtime."""

import os
import sys
from pathlib import Path

import gcp_buildpack as gcp
from common import env, nodejs, ruby, runtime

os_node_version_map = {
    "ubuntu1804": "12.22.12",
    "ubuntu2204": "*",
    "ubuntu2404": "*",
}

def get_rails_node_version(ctx: gcp.Context) -> str:
    """Gets the Node.js version for Rails apps based on the OS."""
    return os_node_version_map[runtime.os_for_stack(ctx)]

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """Detects if the Ruby runtime is needed based on project files."""
    if result := runtime.check_override("ruby"):
        return (result, None)
    
    gemfile_exists = ctx.file_exists("Gemfile")
    if isinstance(gemfile_exists, Exception):
        return (None, gemfile_exists)
    if gemfile_exists:
        return (gcp.OptInFileFound("Gemfile"), None)
    
    gems_rb_exists = ctx.file_exists("gems.rb")
    if isinstance(gems_rb_exists, Exception):
        return (None, gems_rb_exists)
    if gems_rb_exists:
        return (gcp.OptInFileFound("gems.rb"), None)
    
    has_deps, err = ctx.has_at_least_one_outside_dependency_directories("*.rb")
    if err is not None:
        return (None, Exception(f"finding *.rb files: {err}"))
    if not has_deps:
        return (gcp.OptOut("no .rb files found"), None)
    
    return (gcp.OptIn("found .rb files"), None)

def build_fn(ctx: gcp.Context) -> Exception:
    """Builds the Ruby runtime environment."""
    version, err = ruby.detect_version(ctx)
    if err is not None:
        return Exception(f"determining runtime version: {err}")
    
    try:
        layer = ctx.layer("ruby", gcp.LayerType.BUILD | gcp.LayerType.CACHE | 
                        (gcp.LayerType.LAUNCH if not env.is_skip_runtime_launch() else 0))
    except Exception as e:
        return Exception(f"creating layer: {e}")
    
    # Set Node.js version for Rails asset precompilation
    if os.environ.get(nodejs.NODE_VERSION_ENV) == "":
        rails_node_version = get_rails_node_version(ctx)
        ctx.log(f"Setting Node.js runtime version {nodejs.NODE_VERSION_ENV}: {rails_node_version}")
        layer.build_env[nodejs.NODE_VERSION_ENV] = rails_node_version
    
    try:
        installed, err = runtime.install_tarball_if_not_cached(
            ctx, runtime.Runtime.RUBY, version, layer
        )
        if err is not None:
            return err
    except Exception as e:
        return e
    
    # Resolve and store the installed Ruby version
    version_installed, _ = runtime.resolve_version(ctx, runtime.Runtime.RUBY, version, 
                                                  runtime.os_for_stack(ctx))
    layer.build_env[ruby.RUBY_VERSION_KEY] = versionInstalled

    # Check binary dependencies
    try:
        ctx.exec(["ldd", str(Path(layer.path) / "lib/ruby/3.1.0/x86_64-linux/psych.so")])
    except Exception as e:
        return e
    
    # Pin RubyGems and Bundler versions for GAE/GCF compatibility
    if env.is_gae() or env.is_gcf():
        try:
            err = runtime.pin_gem_and_bundler_version(ctx, version, layer)
            if err is not None:
                return Exception(f"updating rubygems and bundler: {err}")
        except Exception as e:
            return e
    
    # Handle tmp/log directories
    local_temp = Path(ctx.application_root()) / "tmp"
    local_log = Path(ctx.application_root()) / "log"
    
    ctx.log("Removing 'tmp' and 'log' directories in user code")
    try:
        if local_temp.exists():
            local_temp.unlink()
        if local_log.exists():
            local_log.unlink()
    except Exception as e:
        return e
    
    # Create symlinks
    try:
        local_temp.symlink_to("/tmp")
        local_log.symlink_to("/var/log")
    except Exception as e:
        return e
    
    return None
