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
Implements php/functions_framework buildpack.
The functions_framework buildpack converts a function into an application and sets up the execution environment.
"""

import os
import json
import subprocess
from pathlib import Path
from typing import Optional, Dict, Any

import google.cloud.buildpacks.gcpbuildpack as gcp
import google.cloud.buildpacks.php as php_utils
import google.cloud.buildpacks.cloudfunctions as cloudfunctions

# Constants
FF_PACKAGE = "google/cloud-functions-framework"
FF_VERSION = "^1.1"
FF_WITH_VERSION = f"{FF_PACKAGE}:{FF_VERSION}"
ROUTER_SCRIPT = "vendor/google/cloud-functions-framework/router.php"
CACHE_TAG = "functions-framework dependencies"

def detect_fn(context: gcp.Context) -> Dict[str, Any]:
    """
    Detects if the functions framework should be used based on environment variables.
    Returns a dictionary indicating whether to opt in or out.
    """
    function_target = os.getenv(env.FUNCTION_TARGET)
    if function_target is not None:
        return {"result": gcp.OPT_IN, "reason": f"Function target {function_target} is set"}
    return {"result": gcp.OPT_OUT, "reason": f"Function target environment variable not set"}

def build_fn(context: gcp.Context) -> None:
    """
    Builds the functions framework environment.
    """
    fn_file = "index.php"
    function_source = os.getenv(env.FUNCTION_SOURCE)
    if function_source is not None:
        fn_file = function_source

    # Syntax check
    try:
        subprocess.run(["php", "-l", fn_file], check=True, cwd=context.application_root)
    except subprocess.CalledProcessError as e:
        raise gcp.BuildpackError(f"Syntax check failed for {fn_file}") from e

    # Handle composer.json cases
    if (context.application_root / "composer.json").exists():
        handle_composer_json(context)
    else:
        handle_no_composer_json(context)

    # Add web process
    context.add_web_process(["php", "-S", "0.0.0.0:${PORT}", ROUTER_SCRIPT])

    # Create and configure layer
    try:
        layer = context.create_layer("functions-framework", gcp.LayerPurpose.BUILD | gcp.LayerPurpose.LAUNCH)
        if not layer:
            raise gcp.BuildpackError("Failed to create functions-framework layer")
        set_functions_env_vars(context, layer)
    except Exception as e:
        raise gcp.BuildpackError(f"Failed to configure layer: {e}") from e

def handle_composer_json(context: gcp.Context) -> None:
    """
    Handles cases where a composer.json file is present.
    """
    try:
        cjs = php_utils.read_composer_json(context.application_root)
    except Exception as e:
        raise gcp.BuildpackError(f"Reading composer.json failed: {e}") from e

    if FF_PACKAGE not in cjs.get("require", {}):
        context.log_info("Handling function without dependency on functions framework")
        cloudfunctions.assert_framework_injection_allowed()
        try:
            php_utils.composer_require(context, [FF_WITH_VERSION])
        except Exception as e:
            raise gcp.BuildpackError(f"Composer require failed: {e}") from e
        cloudfunctions.add_framework_version_label(
            context,
            runtime="php",
            version=FF_VERSION,
            injected=True
        )
    else:
        version = cjs["require"][FF_PACKAGE]
        context.log_info(f"Handling function with dependency on functions framework ({FF_PACKAGE}:{version})")
        cloudfunctions.add_framework_version_label(
            context,
            runtime="php",
            version=version,
            injected=False
       )

def handle_no_composer_json(context: gcp.Context) -> None:
    """
    Handles cases where no composer.json file is present.
    """
    context.log_info("Handling function without composer.json")
    
    vendor_dir = context.application_root / php_utils.VENDOR_DIR
    if not vendor_dir.exists():
        context.log_info("No vendor directory present, installing functions framework")
        converter_dir = Path(__file__).parent.parent / "converter"
        
        # Copy composer files from template
        try:
            subprocess.run(["cp", str(converter_dir / "composer.json"), "."], cwd=context.application_root, check=True)
            subprocess.run(["cp", str(converter_dir / "composer.lock"), "."], cwd=context.application_root, check=True)
        except subprocess.CalledProcessError as e:
            raise gcp.BuildpackError(f"Failed to copy composer files: {e}") from e

        # Install dependencies
        try:
            php_utils.composer_install(context, CACHE_TAG)
        except Exception as e:
            raise gcp.BuildpackError(f"Composer install failed: {e}") from e

        cloudfunctions.add_framework_version_label(
            context,
            runtime="php",
            version=FF_VERSION,
            injected=True
        )
        return

    # Check for existing functions framework in vendor
    ff_path = vendor_dir / FF_PACKAGE.replace("/", os.sep)
    if ff_path.exists():
        context.log_info("Functions framework is already present in the vendor directory")
        
        router_script_path = context.application_root / ROUTER_SCRIPT.lstrip("/")
        if not router_script_path.exists():
            raise gcp.UserError(f"Functions framework router script {ROUTER_SCRIPT} is not present")

        cloudfunctions.add_framework_version_label(
            context,
            runtime="php",
            version="unknown-vendored",
            injected=False
        )
        return

    # Attempt to install functions framework if allowed
    cloudfunctions.assert_framework_injection_allowed()
    
    installed_json = vendor_dir / "composer" / "installed.json"
    if not installed_json.exists():
        raise gcp.UserError(f"{installed_json} is not present, so it appears that Composer was not used to install dependencies.")
    
    context.log_info("Installing functions framework")
    try:
        php_utils.composer_require(context, [FF_WITH_VERSION])
    except Exception as e:
        raise gcp.BuildpackError(f"Failed to install functions framework: {e}") from e

    cloudfunctions.add_framework_version_label(
        context,
        runtime="php",
        version=FF_VERSION,
        injected=True
    )

def set_functions_env_vars(context: gcp.Context, layer: gcp.Layer) -> None:
    """
    Sets environment variables related to functions framework.
    """
    try:
        context.set_env_vars({
            "FUNCTIONS_FRAMEWORK_LAYER": str(layer.path)
        })
    except Exception as e:
        raise gcp.BuildpackError(f"Failed to set environment variables: {e}") from e
