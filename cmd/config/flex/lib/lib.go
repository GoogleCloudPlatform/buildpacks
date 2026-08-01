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

"""Implements config/entrypoint buildpack. The entrypoint buildpack sets the image entrypoint based on an environment variable or Procfile."""

import os
import re
from pathlib import Path
import gcpbuildpack as gcp

env_flex_re = re.compile(r'\s*env\s*:\s*(flex|flexible)\s*')

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception | None]:
    """Detects if the buildpack should be applied."""
    if os.getenv('FLEX_ENV'):
        return (gcp.OptInEnvSet('X_GOOGLE_TARGET_PLATFORM'), None)
    
    yaml_path = os.getenv('GAE_APPLICATION_YAML_PATH')
    if not yaml_path:
        return (gcp.OptOut("Env var GAE_APPLICATION_YAML_PATH is not set, not a GAE Flex app."), None)
    
    full_path = Path(ctx.application_root()) / yaml_path
    if not full_path.exists():
        return (gcp.OptOutFileNotFound(full_path), None)
    
    try:
        content = full_path.read_text()
        if env_flex_re.findall(content):
            return (gcp.OptIn("env: flex found in the application yaml file."), None)
        else:
            return (gcp.OptOut("env: flex not found in the application yaml file."), None)
    except Exception as e:
        return (None, e)

def build_fn(ctx: gcp.Context) -> Exception | None:
    """Builds the layer for the Flex environment."""
    try:
        layer = ctx.layer('flex', gcp.BuildLayer())
        layer.build_environment.default('FLEX_ENV', True)
        return None
    except Exception as e:
        return e
