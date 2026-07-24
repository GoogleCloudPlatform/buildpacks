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
import glob
import shutil
from pathlib import Path

from gcpbuildpack import context, errors
from pkg.nodejs import nodejs
from pkg.ruby import ruby
from pkg.runtime import runtime
from pkg.env import env

const = {
    "node_layer": "node",
    "heapsize_layer": "heapsize",
    "runtime_version_label": "runtime_version"
}

def detect_fn(ctx: context.Context) -> int:
    result = runtime.check_override("nodejs")
    is_rails_app, _ = ruby.needs_rails_asset_precompile(ctx)
    
    if not is_rails_app and result is not None:
        return result

    try:
        pkg_json_exists = ctx.file_exists("package.json")
    except errors.FileError as e:
        raise e
        
    if pkg_json_exists:
        return context.OptInFileFound("package.json")

    try:
        js_files = glob.glob("*.js")
    except Exception as e:
        raise errors.GlobError(f"finding js files: {e}") from e

    if len(js_files) > 0:
        return context.OptIn("found .js files")

    try:
        ts_files = glob.glob("*.ts")
    except Exception as e:
        raise errors.GlobError(f"finding ts files: {e}") from e

    version, err = nodejs.requested_nodejs_version(ctx, None)
    if err:
        raise errors.NodeJSVersionError(f"getting node version: {err}")

    version = version.lstrip("^~")
    version = version.replace("*", "0").replace("x", "0").replace("X", "0")

    try:
        is_node_24_plus = nodejs.version_matches_semver(ctx, ">=24.0.0", version)
    except errors.VersionError as e:
        raise errors.VersionCheckError(f"checking if node version is greater than 24.0.0: {e}") from e

    if len(ts_files) > 0 and is_node_24_plus:
        return context.OptIn("found .ts files")

    return context.OptOut("neither package.json nor any .js files found")

def build_fn(ctx: context.Context) -> None:
    try:
        pjs = nodejs.read_package_json_if_exists(ctx.application_root)
    except errors.FileError as e:
        raise e

    version, err = nodejs.requested_nodejs_version(ctx, pjs)
    if err:
        raise err

    if env.firebase_output_dir in os.environ:
        os_name = runtime.os_for_stack(ctx)
        latest_available_version, err = runtime.resolve_version(ctx, runtime.Nodejs, version, os_name)
        if err:
            raise errors.ResolutionError(f"resolving version {version}: {err}")

        major_version, err = nodejs.major_version(latest_available_version)
        if err:
            raise errors.VersionParseError(f"getting major version for {latest_available_version}: {err}")

        ctx.add_label(const["runtime_version_label"], f"{runtime.Nodejs}{major_version}")

    try:
        nrl = ctx.layer(const["node_layer"], context.BuildLayer, context.CacheLayer, context.LaunchLayerUnlessSkipRuntimeLaunch)
    except errors.LayerError as e:
        raise errors.LayerCreationError(f"creating {const['node_layer']} layer: {e}") from e

    try:
        runtime.install_tarball_if_not_cached(ctx, runtime.Nodejs, version, nrl)
    except errors.InstallationError as e:
        raise errors.RuntimeInstallationError(f"installing nodejs: {e}") from e

    if install_heapsize_script(ctx):
        pass

def install_heapsize_script(ctx: context.Context) -> bool:
    try:
        cap = ctx.capability(execd.installer_capability)
    except errors.CapabilityError as e:
        raise e
        
    if cap is not None:
        return cap.install(ctx, const["heapsize_layer"], "exec/heapsize.sh")

    try:
        l = ctx.layer(const["heapsize_layer"], context.LaunchLayer)
    except errors.LayerError as e:
        raise errors.LayerCreationError(f"creating {const['heapsize_layer']} layer: {e}") from e

    script_path = Path(ctx.buildpack_root) / "exec" / "heapsize.sh"
    dest_path = l.exec.path / "heapsize.sh"

    try:
        data = script_path.read_bytes()
    except FileNotFoundError as e:
        raise errors.FileNotFoundError(f"reading {script_path}: {e}") from e

    os.makedirs(dest_path.parent, exist_ok=True)
    
    try:
        dest_path.write_bytes(data)
    except IOError as e:
        raise errors.WriteError(f"writing {dest_path}: {e}") from e

    return True
