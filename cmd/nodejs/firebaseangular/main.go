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

# Implements nodejs/firebaseangular buildpack.
# The nodejs/firebaseangular buildpack does some prep work for angular and runs the build script.

import os
import json
from pathlib import Path
from packaging.version import Version
import warnings

from gcpbuildpack import Context, DetectResult, OptIn, OptOut, UserError, BuildLayer, CacheLayer
from buildermetadata import GlobalBuilderMetadata, MetadataID, MetadataValue
import nodejs
import apphostingschema
import faherror
import util

MIN_ANGULAR_VERSION = Version("17.2.0")
FRAMEWORK_VERSION_ENV_VAR = "FRAMEWORK_VERSION"

def detect_fn(context: Context) -> tuple[DetectResult, Exception]:
    if not env.is_fah():
        return OptOut("not a firebase apphosting application"), None
    app_dir = util.application_directory(context)
    
    angular_json_path = app_dir / "angular.json"
    if os.path.exists(angular_json_path):
        return OptInFileFound("angular.json"), None
    
    node_deps, err = nodejs.read_node_dependencies(context, app_dir)
    if err:
        return None, err

    apphosting_schema, err = apphostingschema.read_and_validate_from_file(nodejs.APPHOSTING_PREPROCESSED_PATH_FOR_PACK)
    if err:
        return None, err
    
    if nodejs.has_apphosting_package_or_yaml_build(node_deps.package_json, apphosting_schema):
        return OptOut("apphosting build script found"), None

    version = nodejs.version(node_deps, "@angular/core")
    if version:
        return OptIn("angular dependency found"), None
    else:
        return OptOut("angular dependency not found"), None

def build_fn(context: Context) -> Exception:
    app_dir = util.application_directory(context)
    
    node_deps, err = nodejs.read_node_dependencies(context, app_dir)
    if err:
        return err
    
    if not node_deps.lockfile_path:
        return UserError(f"{faherror.MissingLockFileError(app_dir)}")
    
    builder_version = nodejs.version(node_deps, "@angular/core") or node_deps.package_json.dev_dependencies.get("@angular/core", "")
    err = validate_version(context, builder_version)
    if err:
        return err
    
    if version := node_deps.package_json.dependencies.get("@apphosting/adapter-angular"):
        context.log(f"*** Already have @apphosting/adapter-angular@{version}, skipping installation ***")
        context.log("*** Please ensure your build command is set to apphosting-adapter-angular-build ***")
        return None
    
    build_script = node_deps.package_json.scripts.get("build", "")
    if build_script and build_script not in ["ng build", "apphosting-adapter-angular-build"]:
        warnings.warn("*** Custom build command detected, will proceed but may fail due to unexpected output structure ***")

    layer, err = context.layer("npm_modules", BuildLayer | CacheLayer)
    if err:
        return err
    
    if err := nodejs.install_angular_build_adapter(context, layer):
        return err

    layer.build_environment[f"{FRAMEWORK_VERSION_ENV_VAR}"] = builder_version
    GlobalBuilderMetadata().set_value(MetadataID.FrameworkName, "angular")
    GlobalBuilderMetadata().set_value(MetadataID.FrameworkVersion, MetadataValue(builder_version))
    
    adapter_version = context.get_metadata(layer, nodejs.ANGULAR_VERSION_KEY)
    GlobalBuilderMetadata().set_value(MetadataID.AdapterName, "@apphosting/adapter-angular")
    GlobalBuilderMetadata().set_value(MetadataID.AdapterVersion, MetadataValue(adapter_version))

    nodejs.override_angular_build_script(layer)

    return None

def validate_version(context: Context, dep_version: str) -> Exception:
    try:
        version = Version(dep_version)
    except ValueError:
        context.warn(f"Unrecognized angular version: {dep_version}")
        context.warn(f"Consider updating to >= {MIN_ANGULAR_VERSION}")
        return None
    if version < MIN_ANGULAR_VERSION:
        context.warn(f"Update angular dependencies to >= {MIN_ANGULAR_VERSION}")
        return UserError(f"{faherror.UnsupportedFrameworkVersionError('angular', dep_version)}")
    return None
