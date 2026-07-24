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
Implements nodejs/turborepo buildpack.
The nodejs/turborepo buildpack analyzes and configures a build for monorepos using Turborepo.
"""

import json
import os
from typing import Optional, Dict, Any

import gcpbuildpack as gcp
from gcpbuildpack.layers import Layer
from buildermetadata import BuilderMetadata
import nodejs


# Constants
VERSION_KEY = "version"
MONOREPO_PROJECT_ENV_VAR = "MONOREPO_PROJECT"
MONOREPO_COMMAND_ENV_VAR = "MONOREPO_COMMAND"
MONOREPO_BUILD_ARGS_ENV_VAR = "MONOREPO_BUILD_ARGS"


def DetectFn(context: gcp.Context) -> Optional[gcp.DetectResult]:
    """
    Detects if the buildpack should be used based on presence of turbo.json.
    
    Args:
        context (gcp.Context): The build context
        
    Returns:
        Optional[gcp.DetectResult]: Detection result
    """
    turbo_json_path = os.path.join(context.application_root, "turbo.json")
    if not os.path.exists(turbo_json_path):
        return gcp.OptOutFileNotFound("turbo.json")
    
    return gcp.OptInFileFound("turbo.json")


def BuildFn(context: gcp.Context) -> None:
    """
    Builds the application using Turborepo configuration.
    
    Args:
        context (gcp.Context): The build context
    """
    app_dir = os.path.join(context.application_root, "app")
    
    # Read Turbo configuration
    turbo_json_path = os.path.join(context.application_root, "turbo.json")
    turbo_config: Optional[Dict[str, Any]] = None
    if os.path.exists(turbo_json_path):
        with open(turbo_json_path) as f:
            turbo_config = json.load(f)
    
    if not turbo_config:
        raise gcp.UserError("turbo.json file does not exist")
    
    # Read application package.json
    app_package_json_path = os.path.join(app_dir, "package.json")
    app_package: Optional[Dict[str, Any]] = None
    if os.path.exists(app_package_json_path):
        with open(app_package_json_path) as f:
            app_package = json.load(f)
    
    # Determine application name
    app_name = app_package.get("name") if app_package else ""
    if not app_name:
        raise gcp.UserError(
            "Target application in Turbo monorepo is ambiguous. "
            "Please specify the application directory path during onboarding."
        )
    
    # Prepare build arguments
    build_args = [
        f"--filter={app_name}",
        "--env-mode=loose"
    ]
    
    # Create Turbo layer
    turbo_layer = context.layers.create("turbo", gcp.LayerType.BUILD)
    turbo_layer.build_environment[MONOREPO_PROJECT_ENV_VAR] = app_name
    turbo_layer.build_environment[MONOREPO_COMMAND_ENV_VAR] = "turbo"
    turbo_layer.build_environment[MONOREPO_BUILD_ARGS_ENV_VAR] = ",".join(build_args)
    
    # Update builder metadata
    BuilderMetadata().set_value(BuilderMetadata.MONOREPO_NAME, "turbo")
