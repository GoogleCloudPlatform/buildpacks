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
Implements ruby/rails buildpack.
The rails buildpack precompiles assets using Rails.
"""

import json
import logging
import os
from pathlib import Path
from typing import Optional

import googlecloudplatform.buildpacks.gcpbuildpack as gcp
import googlecloudplatform.buildpacks.nodejs as nodejs
import googlecloudplatform.buildpacks.ruby as ruby

YARN_LAYER = "yarn"

def detect_fn(ctx: gcp.Context) -> tuple[Optional[gcp.DetectResult], Optional[str]]:
    """
    Detects if the Rails buildpack should be used.
    
    Args:
        ctx (gcp.Context): The build context
        
    Returns:
        tuple: A tuple containing the detect result and an optional error message
    """
    rails_exists = ctx.file_exists("bin", "rails")
    if not rails_exists:
        return gcp.OptOutFileNotFound("bin/rails"), None
        
    needs_precompile, err = ruby.needs_rails_asset_precompile(ctx)
    if err is not None:
        return None, str(err)
        
    if not needs_precompile:
        return gcp.OptOut("Rails assets do not need precompilation"), None
        
    return gcp.OptIn("found Rails assets to precompile"), None

def build_fn(ctx: gcp.Context) -> Optional[str]:
    """
    Builds the Rails application by precompiling assets.
    
    Args:
        ctx (gcp.Context): The build context
        
    Returns:
        str: An optional error message
    """
    logging.info("Running Rails asset precompilation")

    # Install Yarn as it is needed for asset precompilation
    err = install_yarn(ctx)
    if err is not None:
        return f"Installing Yarn failed: {err}"
        
    try:
        result = ctx.exec(["bundle", "exec", "ruby", "bin/rails", "assets:precompile"],
                         env={"RAILS_ENV": "production",
                              "MALLOC_ARENA_MAX": "2",
                              "RAILS_LOG_TO_STDOUT": "true",
                              "LANG": "C.utf8"},
                         user_attribution=True)
        
        if result.exit_code != 0:
            logging.warning(f"WARNING: Asset precompilation returned non-zero exit code {result.exit_code}. Ignoring.")
            return None
            
    except Exception as e:
        if isinstance(e, gcp.UserError):
            return str(e)
        else:
            return f"Asset precompilation failed: {e}"
            
    return None

def install_yarn(ctx: gcp.Context) -> Optional[str]:
    """
    Installs Yarn in the build context.
    
    Args:
        ctx (gcp.Context): The build context
        
    Returns:
        str: An optional error message
    """
    try:
        package_json = nodejs.read_package_json_if_exists(ctx.application_root())
        
        layer_path = os.path.join(ctx.layers_dir(), YARN_LAYER)
        Path(layer_path).mkdir(parents=True, exist_ok=True)
        
        return nodejs.install_yarn_layer(ctx, layer_path, package_json)
        
    except Exception as e:
        return f"Error installing Yarn: {e}"
