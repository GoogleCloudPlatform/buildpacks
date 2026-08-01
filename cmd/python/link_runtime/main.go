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
Implements python/link-runtime buildpack.
The link-runtime replaces the python layer content installed by the python/runtime buildpack
with symlinks to the python installed in the GAE base images.
"""

import os
import re
import shutil
from pathlib import Path

import packaging.version
from gcpbuildpack import Context, DetectResult, OptOut, InternalError


class lib:
    layer_dir = "/layers/google.python.runtime/python"
    
    link_dirs = [
        "bin",
        "include",
        "lib",
        "share",
    ]

    @staticmethod
    def DetectFn(ctx: Context) -> tuple[DetectResult | None, Exception | None]:
        x_google_skip_runtime_launch = os.getenv("X_GOOGLE_SKIP_RUNTIME_LAUNCH", "")
        
        if not x_google_skip_runtime_launch.lower() == "true":
            return OptOut(f"{x_google_skip_runtime_launch} is not 'true'"), None
            
        # Check for runtime override
        if result := runtime.CheckOverride("python"):
            return result, None

        return OptOut("GOOGLE_RUNTIME env var not a python runtime"), None

    @staticmethod
    def BuildFn(ctx: Context) -> Exception | None:
        try:
            python_path = lib.pythonSystemDir(ctx)
        except Exception as e:
            return InternalError(f"getting python version: {e}")

        for file in lib.link_dirs:
            layer_file = os.path.join(lib.layer_dir, file)
            
            # Remove existing directory
            if os.path.exists(layer_file):
                shutil.rmtree(layer_file)
                
            # Create symlink to system Python directory
            link_path = os.path.join(python_path, file)
            try:
                os.symlink(link_path, layer_file)
            except Exception as e:
                return InternalError(f"creating symlink {link_path} to {layer_file}: {e}")

        return None

    @staticmethod
    def pythonSystemDir(ctx: Context) -> str:
        # Get Python version from context
        ver = ctx.get_python_version()
        
        # Trim 'Python ' prefix and remove RC suffix
        trimmed_ver = lib.versionWithoutRCSuffix(ver.strip("Python "))
        
        # Parse as semver
        try:
            semver = packaging.version.parse(trimmed_ver)
        except Exception as e:
            raise InternalError(f"parsing python version {ver}: {e}")

        return os.path.join("/opt", f"python{semver.major}.{semver.minor}")

    @staticmethod
    def versionWithoutRCSuffix(version: str) -> str:
        # Remove RC suffixes like 'rc1'
        pattern = re.compile(r"rc\d+$", re.IGNORECASE)
        return pattern.sub("", version).rstrip("-")
