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
Implements go/appengine_gomod buildpack.
The appengine_gomod buildpack sets up the path of the package to build for gomod applications.
"""

import os
import shutil
from pathlib import Path

from ..pkg.appengine import opt_out_target_platform_not_gae, opt_out_file_not_found, opt_in
from ..pkg.env import is_gae, buildable_env_var, ga_yaml_main_env_var
from ..pkg.gcpbuildpack import (
    DetectResult,
    BuildLayer,
    GCPContext,
    user_error,
    logf,
)
from ..pkg.golang import build_dir_env

# Constants
stager_file_name = "_main-package-path"

def detect_fn(ctx: GCPContext) -> tuple[DetectResult, Exception]:
    if not is_gae():
        return opt_out_target_platform_not_gae(), None
    
    go_mod_exists = Path("go.mod").exists()
    if not go_mod_exists:
        return opt_out_file_not_found("go.mod"), None

    buildable_path = os.getenv(buildable_env_var)
    if buildable_path is not None:
        return opt_out(f"{buildable_env_var} already defined as {repr(buildable_path)}"), None

    return opt_in(f"found go.mod and {buildable_env_var} is not set"), None

def build_fn(ctx: GCPContext) -> Exception:
    main_path, err = choose_main_path(ctx)
    if err:
        return ValueError(f"choosing main path: {err}")

    clean_path, err = clean_main_path(main_path)
    if err:
        return ValueError(f"cleaning main package path: {err}")

    # Handle main path
    if clean_path != ".":
        build_main_exists = Path(clean_path).exists()
        if build_main_exists:
            clean_path = f"./{clean_path}"
        else:
            logf("Path %r does not exist. Assuming it's a fully qualified package name.", clean_path)

    # Create main_env layer
    main_env_layer, err = ctx.create_layer("main_env", BuildLayer.BUILD)
    if err:
        return ValueError(f"creating main_env layer: {err}")
    main_env_layer.build_environment[buildable_env_var] = clean_path

    # Handle srv layer for backwards compatibility
    srv_layer, err = ctx.create_layer("srv", BuildLayer.BUILD)
    if err:
        return ValueError(f"creating srv layer: {err}")
    srv_layer.build_environment[build_dir_env] = str(srv_layer.path)

    try:
        shutil.copytree(".", srv_layer.path, symlinks=True)
    except Exception as e:
        return e

    return None

def choose_main_path(ctx: GCPContext) -> tuple[str, Exception]:
    ga_main = os.getenv(ga_yaml_main_env_var)
    if ga_main:
        return ga_main, None

    stager_file = ctx.application_root / stager_file_name
    if stager_file.exists():
        try:
            with open(stager_file, "r") as f:
                path = f.read().strip()
            # Clean up the file after reading
            stager_file.unlink()
            return path, None
        except Exception as e:
            return "", e

    return "", None

def clean_main_path(mp: str) -> tuple[str, Exception]:
    if not mp:
        return ".", None
    
    mp = os.path.normpath(mp.replace(os.sep, "/")).strip()
    
    if mp == ".":
        return ".", None
    
    if os.path.isabs(mp):
        return "", user_error(f"main package path {repr(mp)} must not be absolute path")
    
    if mp.startswith(".."):
        return "", user_error(f"main package path {repr(mp)} cannot reference parent")

    return mp, None
