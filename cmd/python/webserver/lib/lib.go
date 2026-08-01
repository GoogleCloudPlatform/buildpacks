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

"""Implements python/webserver buildpack."""

import os
from pathlib import Path
import subprocess

import gcpbuildpack
import python_buildpack_utils as python_utils

layer_name = "gunicorn"
gunicorn_package = "gunicorn"

requirements_content = """\
gunicorn==20.1.0
"""

def detect_fn(ctx: gcpbuildpack.Context) -> tuple[gcpbuildpack.DetectResult, Exception]:
    if os.getenv("Entrypoint") is not None:
        return gcpbuildpack.OptOut("custom entrypoint present"), None

    requirements_path = Path(ctx.work_dir) / "requirements.txt"
    if requirements_path.exists():
        try:
            result = subprocess.run(
                [ctx.python_interpreter, "-m", "pip", "list", "--format=freeze"],
                capture_output=True,
                text=True,
                check=True
            )
            packages = {line.split("==")[0] for line in result.stdout.splitlines()}
            
            if gunicorn_package in packages:
                return gcpbuildpack.OptOut("gunicorn present in requirements.txt"), None
            else:
                return gcpbuildpack.OptIn(
                    "gunicorn missing from requirements.txt",
                    build_plan=python_utils.RequirementsProvidesPlan()
                ), None
        except subprocess.CalledProcessError as e:
            return None, Exception(f"Error detecting gunicorn: {e}")
    else:
        return gcpbuildpack.OptIn(
            "requirements.txt with gunicorn not found", 
            build_plan=python_utils.RequirementsProvidesPlan()
        ), None

def build_fn(ctx: gcpbuildpack.Context) -> Exception:
    try:
        layer = ctx.create_layer(layer_name, gcpbuildpack.LayerType.BUILD)
        
        requirements_file = layer.path / "requirements.txt"
        with open(requirements_file, "w") as f:
            f.write(requirements_content)
            
        ctx.set_build_env_var(
            python_utils.RequirementsFilesEnv,
            str(requirements_file),
            append=True
        )
        return None
    except Exception as e:
        return Exception(f"Error creating {layer_name} layer: {e}")
