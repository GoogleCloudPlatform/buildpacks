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
Implements go/gopath buildpack.
The gopath buildpack downloads dependencies with `go get`.
"""

import os
import shutil
from pathlib import Path

import pkg.gcpbuildpack as gcp
from pkg.fileutil import maybe_copy_path_contents
from pkg.golang import supports_go_get

def detect_fn(context: gcp.Context) -> tuple[gcp.DetectResult, Exception | None]:
    """
    Detects if the buildpack should be used based on presence of go.mod file.
    
    Args:
        context: The build context containing filesystem operations.
        
    Returns:
        A tuple where first element is DetectResult and second is error or None.
    """
    try:
        go_mod_exists = os.path.exists(os.path.join(context.application_root, "go.mod"))
        if go_mod_exists:
            return gcp.OptOut("go.mod found"), None
        return gcp.OptIn("go.mod file not found, assuming GOPATH build"), None
    except Exception as e:
        return None, e

def build_fn(context: gcp.Context) -> Exception | None:
    """
    Builds the application using GOPATH mode.
    
    Args:
        context: The build context containing filesystem operations.
        
    Returns:
        Error or None if successful.
    """
    try:
        layer = context.create_layer("gopath", [gcp.LayerType.BUILD], gcp.LayerType.LAUNCH if context.is_dev_mode else None)
        os.environ["GOPATH"] = layer.path
        os.environ["GO111MODULE"] = "off"

        # Skip 'go get' for Go versions >= 1.22
        if not supports_go_get(context):
            context.log("Skipping go get as it's no longer supported outside of a module in GOPATH mode for Go 1.22+")
            
            vendor_path = os.path.join(context.application_root, "vendor")
            if os.path.exists(vendor_path):
                src_path = os.path.join(layer.path, "src")
                maybe_copy_path_contents(src_path, vendor_path)
            return None

        # Run go get
        result = context.run_command(["go", "get", "-d"], env={"GOPATH": layer.path, "GO111MODULE": "off"})
        if result.returncode != 0:
            raise Exception(f"go get failed with exit code {result.returncode}")
        return None
        
    except Exception as e:
        return e
