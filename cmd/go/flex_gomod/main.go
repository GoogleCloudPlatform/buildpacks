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

import os
import re
import sys
from pathlib import Path

from cmd.go.flex_gomod.gcpbuildpack import (
    BuildLayer,
    Context,
    DetectResult,
    UserError,
)
from cmd.go.flex_gomod.env import (
    BUILDABLE_ENV_VAR,
    FLEX_ENV_VAR,
)

STAGER_FILE_NAME = "_main-package-path"

def detect_fn(ctx: Context) -> tuple[DetectResult, Exception | None]:
    if not is_flex_env():
        return (DetectResult.OptOut("Not a GAE Flex app."), None)
    
    go_mod_exists = ctx.file_exists("go.mod")
    if not go_mod_exists:
        return (DetectResult.FileNotFound("go.mod"), None)
    
    buildable_path = os.getenv(BUILDABLE_ENV_VAR)
    if buildable_path is not None:
        return (DetectResult.OptOut(f"{BUILDABLE_ENV_VAR} already defined as {buildable_path}"), None)
    
    return (DetectResult.OptIn(f"found go.mod and {BUILDABLE_ENV_VAR} is not set"), None)

def build_fn(ctx: Context) -> Exception | None:
    main_path, error = get_main_path(ctx)
    if error:
        return error
    
    cleaned_path, error = clean_main_path(main_path)
    if error:
        return error
    
    if cleaned_path != ".":
        file_exists = ctx.file_exists(cleaned_path)
        if not file_exists:
            ctx.log(f"Path {cleaned_path} does not exist. Assuming it's a fully qualified package name.")
    
    layer, error = ctx.create_layer("main_env", BuildLayer())
    if error:
        return error
    
    layer.build_environment[BUILDABLE_ENV_VAR] = cleaned_path
    return None

def get_main_path(ctx: Context) -> tuple[str, Exception | None]:
    stager_file_path = ctx.application_root / STAGER_FILE_NAME
    if not stager_file_path.exists():
        return ("", None)
    
    try:
        with open(stager_file_path, "r") as f:
            path = f.read().strip()
        
        stager_file_path.unlink()
        return (path, None)
    except Exception as e:
        return ("", e)

def clean_main_path(path: str) -> tuple[str, Exception | None]:
    normalized_path = os.path.normpath(path)
    
    if normalized_path == ".":
        return (".", None)
    
    if os.path.isabs(normalized_path):
        return ("", UserError(f"main package path {path} must not be absolute"))
    
    if re.match(r'^\.\.[/\\]', normalized_path):
        return ("", UserError(f"main package path {path} cannot reference parent"))
    
    return (normalized_path.replace(os.sep, '/'), None)

def is_flex_env() -> bool:
    flex_env = os.getenv(FLEX_ENV_VAR)
    return flex_env and flex_env.lower() == "flex"
