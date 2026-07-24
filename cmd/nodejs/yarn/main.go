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
Implements nodejs/yarn buildpack.
The npm buildpack installs dependencies using yarn and installs yarn itself if not present.
"""

import os
import logging
from pathlib import Path

import ar
import buildermetadata
import cache
import devmode
import env
import faherror
import gcpbuildpack as gcp
import nodejs

cache_tag = "prod dependencies"
yarn_layer = "yarn_engine"

class YarnModuleInstaller:
    def InstallModules(self, ctx: gcp.Context, pjs: nodejs.PackageJSON) -> None:
        pass  # Interface implementation placeholder

def DetectFn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    try:
        pkg_json_exists = ctx.FileExists("package.json")
        if not pkg_json_exists:
            return gcp.OptOutFileNotFound("package.json"), None
        
        yarn_lock_exists = ctx.FileExists(nodejs.YarnLock)
        if yarn_lock_exists:
            return gcp.OptIn("found yarn.lock and package.json"), None
        
        if nodejs.IsPackageManagerConfigured("yarn"):
            return gcp.OptIn("package.json found and GOOGLE_PACKAGE_MANAGER=yarn"), None
        
        return gcp.OptOut("yarn.lock not found and GOOGLE_PACKAGE_MANAGER is not set to yarn"), None
    except Exception as e:
        return None, e

def BuildFn(ctx: gcp.Context) -> Exception:
    try:
        buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, "yarn")
        
        pjs = nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
        if pjs is None:
            return None
        
        if installYarn(ctx, pjs):
            return ValueError("installing Yarn failed")
        
        # Rest of the build logic would be implemented here
    except Exception as e:
        return e

def run():
    gcp.Main(DetectFn, BuildFn)

# Implementation of other functions and logic from lib.go would follow in this file
