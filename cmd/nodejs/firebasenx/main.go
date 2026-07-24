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

"""Implements nodejs/firebasenx buildpack.
The nodejs/firebasenx buildpack analyzes and configures a build for Nx monorepos.
"""

import os
from typing import Dict, List, Optional

import jsonschema  # type: ignore
import schema  # type: ignore
import yaml  # type: ignore

import gcpbuildpack  # type: ignore


version_key = "version"
monorepo_project = "MONOREPO_PROJECT"    # The name of a project in an Nx monorepo.
monorepo_command = "MONOREPO_COMMAND"    # The CLI command utility ("nx").
monorepo_build_args = "MONOREPO_BUILD_ARGS"  # The builder plugin used by the build target executor.
nx_no_cloud = "NX_NO_CLOUD"              # Whether to disable Nx Cloud remote caching.


def detect_fn(ctx: gcpbuildpack.Context) -> Dict:
    """Detect function for the buildpack."""
    if not os.environ.get("X_GOOGLE_TARGET_PLATFORM") == "fah":
        return {"enabled": False, "reason": "not a firebase apphosting application"}
    
    nx_json_exists = ctx.file_exists("nx.json")
    if not nx_json_exists:
        return {"enabled": False, "reason": "nx.json file not found"}
        
    return {"enabled": True, "reason": "nx.json file found"}


def build_fn(ctx: gcpbuildpack.Context) -> None:
    """Build function for the buildpack."""
    app_dir = ctx.get_application_directory()
    
    nx_json = read_nx_json(app_dir)
    if not nx_json:
        raise ValueError("nx.json file does not exist")
        
    project_name = nx_json.get("defaultProject", "")
    if not project_name:
        raise ValueError("target application in Nx monorepo is ambiguous. Please specify the application directory path during onboarding or a default project in nx.json")
        
    build_args = [f"--project={project_name}"]
    
    layer = ctx.create_layer("nx")
    layer.build_environment[monorepo_project] = project_name
    layer.build_environment[monorepo_command] = "nx"
    layer.build_environment[monorepo_build_args] = ",".join(build_args)
    layer.build_environment[nx_no_cloud] = "true"
    
    ctx.add_builder_metadata(buildermetadata.MonorepoName, "nx")
