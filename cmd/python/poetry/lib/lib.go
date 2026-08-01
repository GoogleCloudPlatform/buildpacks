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
Implements python/poetry buildpack.
The poetry buildpack installs dependencies using poetry.
"""

import logging
from typing import Any, Optional

import buildpack.buildermetadata as buildermetadata
import buildpack.buildermetrics as buildermetrics
import buildpack.gcp as gcp
import buildpack.python as python

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Optional[Exception]]:
    """
    Detects if the project is a poetry project.
    
    Args:
        ctx: The build context
        
    Returns:
        A tuple containing the detection result and any error encountered
    """
    try:
        is_poetry, message = python.is_poetry_project(ctx)
        if is_poetry:
            return gcp.OptIn(message), None
        return gcp.OptOut(message), None
    except Exception as e:
        return gcp.OptOut(str(e)), e

def build_fn(ctx: gcp.Context) -> Optional[Exception]:
    """
    Builds the poetry project.
    
    Args:
        ctx: The build context
        
    Returns:
        Any error encountered during the build process, or None if successful
    """
    try:
        # Increment metrics and set metadata
        buildermetrics.global_builder_metrics().get_counter(buildermetrics.PoetryUsageCounterID).increment(1)
        buildermetadata.global_builder_metadata().set_value(buildermetadata.PackageManager, "poetry")
        buildermetadata.global_builder_metadata().set_value(buildermetadata.ConfigFile, "pyproject.toml")

        # Install Poetry
        if error := python.install_poetry(ctx):
            return Exception(f"Installing poetry failed: {error}")

        # Ensure poetry.lock exists or generate it
        if error := python.ensure_poetry_lockfile(ctx):
            return Exception(f"Ensuring poetry.lock failed: {error}")

        # Install dependencies and configure the environment
        if error := python.poetry_install_dependencies_and_configure_env(ctx):
            return Exception(f"Installing dependencies and configuring env failed: {error}")

        # Check for incompatible dependencies
        result = ctx.exec_command(["poetry", "check"], user_attribution=True)
        if result.error:
            logging.warning("Warning: 'poetry check' returned an error, which might just be a deprecation warning: %s", result.error)
        else:
            logging.debug("No incompatible dependencies found.")

        return None
    except Exception as e:
        return e
