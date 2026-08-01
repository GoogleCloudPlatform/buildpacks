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

"""Implements python/pip buildpack. The pip buildpack installs dependencies using pip."""

import os
import sys
from dataclasses import dataclass
from typing import Any, Optional

import buildermetadata
import buildermetrics
import gcpbuildpack as gcp
import libcnb
import python_utils

const_layer_name = "pip"

@dataclass
class Metadata:
    """Metadata stored for a dependencies layer."""
    python_version: str = ""
    dependency_hash: str = ""
    expiry_timestamp: str = ""

def detect_fn(ctx: gcp.Context) -> tuple[Optional[gcp.DetectResult], Optional[str]]:
    """
    Detect function for pip buildpack.
    
    Args:
        ctx: The context containing build information.
        
    Returns:
        A tuple with the detection result and any error encountered.
    """
    if python_utils.is_pip_pyproject(ctx):
        return gcp.OptIn(f"found pyproject.toml, using pip because {python_utils.env.PythonPackageManager} is set to 'pip'"), None
    
    plan = libcnb.BuildPlan(requires=python_utils.RequirementsRequires)
    
    requirements_exists = ctx.file_exists("requirements.txt")
    if isinstance(requirements_exists, Exception):
        return None, str(requirements_exists)
        
    if requirements_exists:
        plan.provides = python_utils.RequirementsProvides
        
    return gcp.OptInAlways(gcp.WithBuildPlans(plan)), None

def build_fn(ctx: gcp.Context) -> Optional[str]:
    """
    Build function for pip buildpack.
    
    Args:
        ctx: The context containing build information.
        
    Returns:
        Any error encountered during the build process, or None if successful.
    """
    buildermetrics.GlobalBuilderMetrics().get_counter(buildermetrics.PIPUsageCounterID).increment(1)
    buildermetadata.GlobalBuilderMetadata().set_value(buildermetadata.PackageManager, "pip")

    try:
        layer = ctx.layer(const_layer_name, gcp.BuildLayer | gcp.CacheLayer | gcp.LaunchLayer)
    except Exception as err:
        return f"creating {const_layer_name} layer: {err}"
    
    if python_utils.is_pip_pyproject(ctx):
        buildermetadata.GlobalBuilderMetadata().set_value(buildermetadata.ConfigFile, "pyproject.toml")
        
        try:
            python_utils.pip_install_pyproject(ctx, layer)
        except Exception as err:
            return f"installing dependencies from pyproject.toml: {err}"
    else:
        buildermetadata.GlobalBuilderMetadata().set_value(buildermetadata.ConfigFile, "requirements.txt")
        
        reqs_env = os.getenv(python_utils.RequirementsFilesEnv, "")
        reqs = list(filter(None, reqs_env.split(os.pathsep)))
        ctx.debug(f"Found requirements.txt files provided by other buildpacks: {reqs}")
        
        if ctx.file_exists("requirements.txt"):
            reqs.append("requirements.txt")
            
        try:
            python_utils.pip_install_requirements(ctx, layer, *reqs)
        except Exception as err:
            return f"installing dependencies from requirements.txt and validating them: {err}"
    
    return None
