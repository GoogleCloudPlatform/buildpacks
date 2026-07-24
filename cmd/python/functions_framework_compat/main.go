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
Implements python/functions_framework_compat buildpack.
The functions_framework buildpack installs dependencies that were included with the python37 runtime.
"""

import os
import logging

from google.cloud.functions import BuildContext
from ...pkg.env import (
    env,
    is_gcf,
)
from ...pkg.gcpbuildpack import (
    OptInResult,
    OptOutResult,
    DetectResult,
    BuildPlan,
)

const_layer_name = "functions-framework-compat"

def detect_fn(context: BuildContext) -> tuple[DetectResult, Exception]:
    """
    Detection function for the buildpack.
    Checks if the environment is GCF and runtime is python37.
    Returns OptIn or OptOut results based on conditions.
    """
    if not is_gcf():
        return OptOutResult("Deployment environment is not GCF."), None
    
    runtime = os.getenv(env.Runtime)
    if runtime != "python37":
        return OptOutResult(f"env var {env.Runtime} is not set to python37"), None
    
    function_target = os.getenv(env.FunctionTarget)
    if function_target:
        plan = BuildPlan()
        plan.add_requirement("python.RequirementsProvidesPlan")
        return OptInResult([plan]), None
    else:
        return OptOutResult(f"env var {env.FunctionTarget} not set"), None


def build_fn(context: BuildContext) -> Exception:
    """
    Build function for the buildpack.
    Creates layers and sets up environment variables.
    """
    # Create layer
    layer = context.create_layer(const_layer_name, is_launch=True)
    
    # Add requirements file
    req_file_path = os.path.join(
        context.buildpack_root,
        "converter",
        "requirements.txt"
    )
    logging.debug("Adding functions-framework requirements.txt to the list of requirements files to install.")
    layer.add_to_build_env(python.RequirementsFilesEnv, req_file_path)
    
    # Set entry point
    function_target = os.getenv(env.FunctionTarget)
    if function_target:
        layer.set_launch_env("ENTRY_POINT", function_target)
    
    return None
