# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Implements python/uv buildpack.
The uv buildpack installs dependencies using uv.
"""

import os
import subprocess
from pathlib import Path

import buildermetrics
import buildermetadata
import gcpbuildpack as gcp
import libcnb


layer_name = "uv-dependencies"


def is_uv_pyproject(ctx):
    """
    Determines if the project uses UV via pyproject.toml.

    Args:
        ctx: The build context.

    Returns:
        bool, str, error: Whether it's a UV project, message, and error.
    """
    try:
        # Read pyproject.toml
        pyproject_path = Path(ctx.work_dir) / "pyproject.toml"
        if not pyproject_path.exists():
            return False, "No pyproject.toml found", None

        with open(pyproject_path, 'r') as f:
            content = f.read()
            if "[build-system]" in content and "uv" in content.lower():
                return True, "Using UV via pyproject.toml", None
    except Exception as e:
        return False, f"Error reading pyproject.toml: {e}", e

    return False, "Not a UV project via pyproject.toml", None


def is_uv_requirements(ctx):
    """
    Determines if the project uses UV via requirements.txt.

    Args:
        ctx: The build context.

    Returns:
        bool, str, error: Whether it's a UV project, message, and error.
    """
    try:
        # Read requirements.txt
        reqs_path = Path(ctx.work_dir) / "requirements.txt"
        if not reqs_path.exists():
            return False, "No requirements.txt found", None

        with open(reqs_path, 'r') as f:
            content = f.read()
            if any(line.strip().lower() == "uv" for line in content.split('\n')):
                return True, "Using UV via requirements.txt", None
    except Exception as e:
        return False, f"Error reading requirements.txt: {e}", e

    return False, "Not a UV project via requirements.txt", None


def detect_fn(ctx):
    """
    Detect function for the buildpack.

    Args:
        ctx: The build context.

    Returns:
        libcnb.DetectResult or None
    """
    is_uv, message, err = is_uv_pyproject(ctx)
    if err:
        return gcp.OptOut(message), err

    if is_uv:
        return gcp.OptIn(message), None

    plan = libcnb.BuildPlan(Requires=python.RequirementsRequires)
    
    # Check for requirements.txt
    reqs_path = Path(ctx.work_dir) / "requirements.txt"
    if reqs_path.exists():
        plan.Provides = python.RequirementsProvides

    is_uv_req, message, err = is_uv_requirements(ctx)
    if err:
        return gcp.OptOut(message), err

    if is_uv_req:
        return gcp.OptIn(message, gcp.WithBuildPlans(plan)), None

    return gcp.OptOut(message), None


def build_fn(ctx):
    """
    Build function for the buildpack.

    Args:
        ctx: The build context.
    """
    try:
        # Increment usage counter
        buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.UVUsageCounterID).Increment(1)
        
        # Set package manager metadata
        buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, "uv")

        # Install UV
        subprocess.run(["pip", "install", "uv"], check=True)

        # Create layer
        layer = ctx.Layer(layer_name, gcp.BuildLayer | gcp.CacheLayer | gcp.LaunchLayer)
        
        is_uv, _, err = is_uv_pyproject(ctx)
        if err:
            raise err

        if is_uv:
            buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.ConfigFile, "pyproject.toml")
            # Install dependencies
            subprocess.run(["uv", "pip", "install", "-r", "pyproject.toml"], check=True)
        else:
            buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.ConfigFile, "requirements.txt")
            
            # Get requirements files
            reqs = os.getenv(python.RequirementsFilesEnv, "").split(os.pathsep)
            if not reqs:
                reqs = []
                
            # Add workspace requirements.txt if exists
            if (Path(ctx.work_dir) / "requirements.txt").exists():
                reqs.append("requirements.txt")
                
            ctx.Log(f"Found requirements.txt, installing with `uv pip install`.")
            
            # Install requirements
            subprocess.run(["uv", "pip", "install"] + reqs, check=True)

    except Exception as e:
        raise gcp.UserError(f"Error building with UV: {e}") from e


def gcp_main(detect_fn, build_fn):
    """
    Main function that follows GCP buildpack conventions.

    Args:
        detect_fn: The detection function.
        build_fn: The build function.
    """
    # This would be integrated with the GCP buildpack runtime
    pass  # Replace with actual implementation as needed
