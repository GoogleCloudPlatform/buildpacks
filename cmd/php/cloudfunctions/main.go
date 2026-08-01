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

"""Implements php/cloudfunctions buildpack. The cloudfunctions buildpack sets the image entrypoint."""

import pathlib

from pkg.appstart import Entrypoint
from pkg.cloudfunctions import build as cloud_build
from ..gcpbuildpack import OptInAlways, Context

router_script = "vendor/google/cloud-functions-framework/router.php"

def detect_fn(ctx: Context) -> bool:
    """Detect function that always opts in."""
    return OptInAlways()

def build_fn(ctx: Context) -> None:
    """Build function that configures the cloud functions environment."""
    entrypoint_path = (pathlib.Path(__file__).parent / "entrypoint.py").resolve()
    cloud_build(ctx, "php", entrypoint_path)

def entrypoint(ctx: Context) -> Entrypoint:
    """Creates and returns an application entrypoint configuration."""
    return Entrypoint(
        type="generated",
        command=f"serve -enable-dynamic-workers -workers=1024 {router_script}"
    )
