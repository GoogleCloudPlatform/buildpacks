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

"""Implements go/appengine buildpack. The appengine buildpack sets the image entrypoint."""

import os
import subprocess
from typing import List, Optional

import gcp_buildpack as gcp
from packages.appengine import AppEngineBuildpack


def detect_fn(context: gcp.Context) -> tuple[int, Optional[Exception]]:
    """Detect function for the appengine buildpack."""
    if os.getenv("GAE_ENV"):
        return (AppEngineBuildpack.OPT_IN_TARGET_PLATFORM_GAE, None)
    return (AppEngineBuildpack.OPT_OUT_TARGET_PLATFORM_NOT_GAE, None)


def build_fn(context: gcp.Context) -> Optional[Exception]:
    """Build function for the appengine buildpack."""
    if error := validate_app_engine_apis(context):
        return error
    return AppEngineBuildpack.build(
        context, "go", entrypoint
    )


def validate_app_engine_apis(context: gcp.Context) -> Optional[Exception]:
    """Validates usage of App Engine APIs."""
    supportsApis = golang_supports_appengine_apis(context)
    if not supportsApis:
        direct_deps = get_direct_deps(context)
        if app_engine_in_deps(direct_deps):
            context.warn(AppEngineBuildpack.APP_ENGINE_WARNING)
            return None

    all_deps = get_all_deps(context)
    usingAppEngine = app_engine_in_deps(all_deps)
    
    if supportsApis and not usingAppEngine:
        context.warn(AppEngineBuildpack.APP_ENGINE_UNUSED_API_WARNING)
        
    if not supportsApis and usingAppEngine:
        context.warn(AppEngineBuildpack.APP_ENGINE_INDIRECT_DEP_WARNING)
        
    return None


def entrypoint(context: gcp.Context) -> dict:
    """Generates the entrypoint configuration."""
    context.log(f"No user entrypoint specified. Using the generated entrypoint {golang.OUT_BIN}")
    return {
        "type": AppEngineBuildpack.EntrypointType.GENERATED.value,
        "command": golang.OUT_BIN
    }


def app_engine_in_deps(deps: List[str]) -> bool:
    """Checks if any dependency is related to App Engine."""
    for dep in deps:
        if dep.startswith("google.golang.org/appengine"):
            return True
    return False


def get_all_deps(context: gcp.Context) -> List[str]:
    """Retrieves all dependencies using go list."""
    result = context.exec(["go", "list", "-e", "-f", "{{join .Deps \"\\n\"}}", "./..."])
    if result.error:
        return []
    return result.stdout.strip().split()


def get_direct_deps(context: gcp.Context) -> List[str]:
    """Retrieves direct dependencies using go list."""
    result = context.exec(["go", "list", "-e", "-f", "{{join .Imports \"\\n\"}}", "./..."])
    if result.error:
        return []
    return result.stdout.strip().split()
