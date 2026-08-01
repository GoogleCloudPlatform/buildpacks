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
Implements ruby/appengine buildpack.
The appengine buildpack sets the image entrypoint.
"""

import os
import logging
from pathlib import Path

import gcpbuildpack as gcp
from gcpbuildpack import appengine, appstart, env
import ruby  # Assuming this is a Python package equivalent to the Go ruby package

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception | None]:
    """
    Determines if the buildpack should be applied based on GAE environment.
    
    Args:
        ctx: The build context
        
    Returns:
        Tuple containing DetectResult and possible error
    """
    if env.is_gae():
        return appengine.OPT_IN_TARGET_PLATFORM_GAE, None
    return appengine.OPT_OUT_TARGET_PLATFORM_NOT_GAE, None

def build_fn(ctx: gcp.Context) -> Exception | None:
    """
    Builds the application with necessary symlinks and entrypoint configuration.
    
    Args:
        ctx: The build context
        
    Returns:
        Possible error or None
    """
    # Ruby sometimes writes to local directories tmp/ and log/, so we link these to writable areas.
    local_temp = Path(ctx.application_root) / "tmp"
    local_log = Path(ctx.application_root) / "log"

    try:
        if local_temp.exists():
            local_temp.unlink()
        local_temp.symlink_to("/tmp")

        if local_log.exists():
            local_log.unlink()
        local_log.symlink_to("/var/log")
    except Exception as e:
        return e

    # Build with entrypoint
    result = appengine.build(
        ctx,
        "ruby",
        lambda ctx, src_dir: entrypoint(ctx, str(src_dir))
    )
    
    return None if result else result  # Assuming build returns success status

def entrypoint(ctx: gcp.Context, src_dir: str) -> tuple[appstart.Entrypoint, Exception | None]:
    """
    Infers the application's entrypoint and configures it.
    
    Args:
        ctx: The build context
        src_dir: Source directory path
        
    Returns:
        Tuple containing Entrypoint object or error
    """
    try:
        ep = ruby.infer_entrypoint(ctx, src_dir)
        if not ep:
            raise ValueError("Could not infer entrypoint")
            
        logging.warning(
            "WARNING: No entrypoint specified. Attempting to infer entrypoint, but it is recommended "
            "to set an explicit `entrypoint` in app.yaml."
        )
        
        logging.info(f"Using inferred entrypoint: {ep!r}")
        
        return appstart.Entrypoint(
            type=appstart.EntrypointGenerated,
            command=ep
        ), None
        
    except Exception as e:
        return None, e
