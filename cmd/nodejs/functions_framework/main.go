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

"""Implements nodejs/functions_framework buildpack library."""

import os
import sys
from dataclasses import dataclass
from typing import Optional

import google_cloud_platform.buildpacks.gcpbuildpack as gcp
import google_cloud_platform.buildpacks.pkg.nodejs as nodejs
import google_cloud_platform.buildpacks.pkg.env as env
import google_cloud_platform.buildpacks.pkg.cloudfunctions as cloudfunctions
import google_cloud_platform.buildpacks.pkg.cache as cache

class Layer:
    def __init__(self, name: str):
        self.name = name
        self.path = os.path.join(os.getcwd(), "layers", name)
        self.launch_environment = LaunchEnvironment()

@dataclass
class LaunchEnvironment:
    def prepend(self, key: str, sep: str, value: str) -> None:
        existing = os.getenv(key, "")
        new_value = f"{value}{sep}{existing}" if existing else value
        os.environ[key] = new_value

    def default(self, key: str, value: str) -> None:
        if not os.getenv(key):
            os.environ[key] = value

def DetectFn(context: gcp.Context) -> tuple[Optional[gcp.DetectResult], Optional[Exception]]:
    """Detect function for nodejs functions framework."""
    if nodejs.is_nodejs8_runtime():
        return gcp.OptOut("Incompatible with nodejs8"), None
    if os.getenv(env.FUNCTION_TARGET):
        return gcp.OptInEnvSet(env.FUNCTION_TARGET), None
    return gcp.OptOutEnvNotSet(env.FUNCTION_TARGET), None

def BuildFn(context: gcp.Context) -> Optional[Exception]:
    """Build function for nodejs functions framework."""
    if os.getenv(env.FUNCTION_SOURCE):
        return gcp.UserError(f"{env.FUNCTION_SOURCE} is not currently supported for Node.js buildpacks")

    index_js_exists = context.file_exists("index.js")
    fn_file = "function.js"
    if index_js_exists:
        fn_file = "index.js"

    has_framework_dependency = False
    package_json = nodejs.read_package_json_if_exists(context.application_root())
    if package_json:
        functions_framework_pkg = "@google-cloud/functions-framework"
        has_framework_dependency = functions_framework_pkg in package_json.dependencies
        if package_json.main:
            fn_file = package_json.main

    fn_file_exists = context.file_exists(fn_file)
    if not fn_file_exists:
        return gcp.UserError(f"{fn_file} does not exist")

    # Rest of the build logic continues here...
