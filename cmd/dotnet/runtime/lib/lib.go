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
Binary dotnet/runtime buildpack detects .NET applications
and install the corresponding version of .NET runtime.
"""

import os
import pathlib
from typing import Optional, Dict

import gcpbuildpack
import libcnb

runtime_layer_name = "runtime"
version_key = "version"

def detect_fn(ctx: gcpbuildpack.Context) -> tuple[Optional[gcpbuildpack.DetectResult], Exception]:
    if result := runtime.check_override("dotnet"):
        return (result, None)
    
    files, err = dotnet.project_files(ctx, ".")
    if err:
        return (None, err)
    if len(files) > 0:
        return (gcpbuildpack.OptIn(f"found project files: {', '.join(files)}"), None)
    
    rt_cfgs, err = dotnet.runtime_config_json_files(".")
    if err:
        return (None, Exception(f"finding runtimeconfig.json: {err}"))
    if len(rt_cfgs) > 0:
        return (gcpbuildpack.OptIn("found at least one runtimeconfig.json"), None)
    
    return (gcpbuildpack.OptOut("no project files or .dll files found"), None)

def build_fn(ctx: gcpbuildpack.Context) -> Exception:
    is_dev_mode, err = env.is_dev_mode()
    if err:
        return Exception(f"checking if dev mode is enabled: {err}")
    if is_dev_mode:
        return None
    
    runtime_version, err = dotnet.get_runtime_version(ctx, ctx.application_root)
    if err:
        return Exception(f"getting runtime version: {err}")
    if error := build_runtime_layer(ctx, runtime_version):
        return Exception(f"building the runtime layer: {error}")
    return None

def build_runtime_layer(ctx: gcpbuildpack.Context, rt_version: str) -> Optional[Exception]:
    rtl, err = ctx.layer(runtime_layer_name, gcpbuildpack.CacheLayer, gcpbuildpack.LaunchLayer)
    if err:
        return Exception(f"creating {runtime_layer_name} layer: {err}")
    
    if error := runtime.install_tarball_if_not_cached(ctx, runtime.asp_net_core, rt_version, rtl):
        return error
    
    ctx.add_installed_runtime_version(rt_version)

    cap = ctx.capability(dotnet.skip_env_variables_assignment_capability)
    if cap:
        if isinstance(cap, dotnet.SkipEnvVariablesAssignment):
            return cap.skip_variables(ctx, rtl)
        else:
            return gcpbuildpack.InternalError(f"capability {dotnet.skip_env_variables_assignment_capability} must implement dotnet.SkipEnvVariablesAssignment")
    
    set_runtime_env_vars(ctx, rtl)
    return None

def set_runtime_env_vars(ctx: gcpbuildpack.Context, rtl: libcnb.Layer) -> None:
    rtl.launch_environment.default("DOTNET_ROOT", str(rtl.path))
    rtl.launch_environment.prepend("PATH", os.pathsep, str(rtl.path))
    rtl.launch_environment.default("DOTNET_RUNNING_IN_CONTAINER", "true")
    if dotnet.requires_globalization_invariant(ctx):
        rtl.launch_environment.default("DOTNET_SYSTEM_GLOBALIZATION_INVARIANT", "1")
