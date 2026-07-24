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

"""Implements ruby/missing-entrypoint buildpack."""

import os
from googlecloudplatform.buildpacks.gcpbuildpack import context as gcp_context
from googlecloudplatform.buildpacks.pkg import env, runtime

def detect(ctx: gcp_context.Context) -> bool:
    """Detect phase for the buildpack."""
    # Check if this buildpack has been overridden
    if 'ruby' in ctx.runtime_overrides:
        return False  # Opt out
    
    try:
        has_files = ctx.has_at_least_one("*.rb")
        if not has_files:
            return False  # No .rb files found, opt out
        return True  # Found .rb files, opt in
    except Exception as e:
        raise RuntimeError(f"Error detecting files: {e}") from e

def build(ctx: gcp_context.Context) -> None:
    """Build phase for the buildpack."""
    error_msg = (
        "For Ruby, an entrypoint must be manually set, either with the %s env var "
        "or by creating a %s file."
    )
    raise ValueError(error_msg % (env.Entrypoint, "Procfile"))
