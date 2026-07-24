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

"""Implements dotnet/appengine buildpack. The appengine buildpack sets the image entrypoint."""

import os

from ...pkg.appengine import OptInTargetPlatformGAE, OptOutTargetPlatformNotGAE
from ...pkg.gcpbuildpack import Context, DetectResult, UserError


def DetectFn(ctx: Context) -> tuple[DetectResult, Exception | None]:
    """Detects if the environment is Google App Engine."""
    if os.getenv("GAE"):
        return OptInTargetPlatformGAE(), None
    return OptOutTargetPlatformNotGAE(), None


def BuildFn(ctx: Context) -> Exception | None:
    """Build function that sets up the entrypoint for the application."""
    return appengine_build(ctx, "dotnet", entrypoint)


def entrypoint(ctx: Context) -> tuple[dict, Exception | None]:
    """Determines the entrypoint command from environment variables."""
    ep = os.getenv("ENTRYPOINT")
    if not ep:
        return None, UserError("Expected entrypoint from app.yaml or root project file, found nothing")
    
    ctx.log(f"Using the entrypoint: {ep}")
    return {"type": "generated", "command": ep}, None
