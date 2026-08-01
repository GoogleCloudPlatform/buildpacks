# Copyright 2026 Google LLC
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
Package lib provides the buildpack logic for serving static sites.
"""

import os
from pathlib import Path
import logging

import env
import gcp  # Assuming this is imported from the appropriate package
import static

class Constants:
    INDEX_HTML = "index.html"
    NGINX_PATH_BASE_IMAGE = "/etc/nginx/"
    NGINX_PATH_BUILDPACKS = "/layers/google.utils.nginx/nginx/conf"

STATIC_ASSETS = [
    "build",
    "dist",
    "public",
    "_site",
    "site",
    Constants.INDEX_HTML,
]

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """
    Detects static assets in the application directory.
    
    Args:
        ctx: The buildpack context
        
    Returns:
        A tuple containing the detection result and any error
    """
    # Restrict this feature behind ALPHA release track.
    if not env.is_alpha_supported():
        return gcp.OptOut("Static runtimes feature is supported only on ALPHA release track."), None

    for asset in STATIC_ASSETS:
        full_path = Path(ctx.application_root()) / asset
        try:
            info = full_path.stat()
            if asset == Constants.INDEX_HTML and not info.is_dir():
                return gcp.OptInFileFound(asset), None
            if asset != Constants.INDEX_HTML and info.is_dir():
                return gcp.OptInFileFound(asset), None
        except FileNotFoundError:
            continue

    return gcp.OptOut("No static asset folders or index.html found."), None

def build_fn(ctx: gcp.Context) -> Exception:
    """
    Builds the nginx configuration for serving static assets.
    
    Args:
        ctx: The buildpack context
        
    Returns:
        Any error that occurred during building
    """
    try:
        layer = ctx.layer("nginx_config", gcp.LayerType.LAUNCH, gcp.LayerType.BUILD)
    except Exception as e:
        return Exception(f"creating layer: {e}")

    layer.build_environment.override(env.StaticServe, "true")

    root_path = Path(ctx.application_root())
    for asset in STATIC_ASSETS:
        full_path = root_path / asset
        if not full_path.exists():
            continue
            
        info = full_path.stat()
        if asset == Constants.INDEX_HTML and not info.is_dir():
            ctx.log(f"Target static asset found: index.html at root.")
            break
        if asset != Constants.INDEX_HTML and info.is_dir():
            root_path = full_path
            ctx.log(f"Target static asset folder found: {asset}")
            break

    nginx_conf_path = layer.path / static.NGINX_CONF_FILE
    ctx.log(f"Generating default SPA/SSG-friendly {static.NGINX_CONF_FILE}")

    if env.is_static_base_image():
        nginx_path = Constants.NGINX_PATH_BASE_IMAGE
    else:
        nginx_path = Constants.NGINX_PATH_BUILDPACKS

    nginx_mime_types_path = Path(nginx_path) / "mime.types"

    params = static.NginxConfigParams(
        root_path=root_path,
        mime_types_path=nginx_mime_types_path
    )

    try:
        static.write_nginx_config(nginx_conf_path, params)
    except Exception as e:
        return Exception(f"writing {static.NGINX_CONF_FILE}: {e}")

    # Setup Entrypoint
    ctx.add_process(
        gcp.ProcessType.WEB,
        [
            "nginx",
            "-p", str(nginx_path),
            "-c", str(nginx_conf_path),
            "-g", "daemon off;"
        ],
        default_process=True,
        direct_process=True
    )

    return None
