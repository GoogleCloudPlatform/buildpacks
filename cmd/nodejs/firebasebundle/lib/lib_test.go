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
import yaml
from pathlib import Path
from typing import Optional, Dict, List

import gcpbuildpack as gcp
from ..buildermetadata import BuilderMetadata
from ..nodejs import PackageJSON

def detect_fn(context: gcp.Context) -> gcp.DetectResult:
    if not env.is_fah():
        return gcp.OptOut("not a firebase apphosting application")
    
    use_generic, err = env.is_present_and_true(env.GOOGLE_USE_GENERIC_FIREBASEBUNDLE)
    if err is not None:
        context.warn(f"failed to parse {env.GOOGLE_USE_GENERIC_FIREBASEBUNDLE}: {err}")
    
    if use_generic:
        return gcp.OptOut("not using google.nodejs.firebasebundle because GOOGLE_USE_GENERIC_FIREBASEBUNDLE is true")
    
    return gcp.OptIn("firebase apphosting application")

def build_fn(context: gcp.Context) -> None:
    bundle_path = context.application_root / ".apphosting" / "bundle.yaml"
    bundle_yaml, err = read_bundle_yaml(context, bundle_path)
    if err is not None:
        raise err

    output_bundle_dir = os.environ.get("FIREBASE_OUTPUT_BUNDLE_DIR")
    if not output_bundle_dir:
        raise gcp.InternalError(f"environment variable {output_bundle_dir} not found")

    workspace_public_dir = context.application_root / "public"
    output_public_dir = Path(output_bundle_dir) / "public"

    app_dir = util.application_directory(context)

    apphosting_yaml_path_tests = os.environ.get("APPHOSTINGYAML_FILEPATH_TESTS")
    if apphosting_yaml_path_tests:
        apphosting_yaml, err = read_app_hosting_yaml(context, apphosting_yaml_path_tests)
    else:
        apphosting_yaml, err = read_app_hosting_yaml(context, "/workspace/apphosting_preprocessed")

    if err is not None:
        raise err

    if bundle_yaml is None:
        context.log("bundle.yaml does not exist, assuming default configs")
        err = generate_default_bundle_yaml(bundle_path, context)
        if err is not None:
            raise err

    context.log("Copying static assets.")
    try:
        fileutil.copy_file(context, output_bundle_dir / "bundle.yaml", bundle_path)
    except Exception as e:
        raise gcp.InternalError(f"copying output bundle dir {output_bundle_dir}: {e}")

    if not copy_public_dir_to_output Bundle Dir(output_public_dir, workspace_public_dir, context):
        pass  # Handle error

    node_deps, err = nodejs.read_node_dependencies(context, app_dir)
    if err is None:
        set_metadata(node_deps.package_json)

    err = delete_files_not_included(context, apphosting_yaml, bundle_yaml, context.application_root)
    if err is not None:
        raise err

    err = set_run_command(apphosting_yaml, bundle_yaml, context)
    if err is not None:
        raise err
