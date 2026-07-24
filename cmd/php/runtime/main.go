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

"""Implements php/runtime buildpack. The runtime buildpack installs the PHP runtime."""

import os
from pathlib import Path
import logging

import gcpbuildpack as gcp  # type: ignore
import libcnb  # type: ignore
import php_packaging  # type: ignore
import runtime_pkg as runtime  # type: ignore


PHP_INI_NAME = "php.ini"

def detect_fn(context: gcp.Context) -> dict:
    """Detect if PHP runtime should be used."""
    override_result = runtime.check_override("php")
    if override_result:
        return override_result
    
    composer_json_path = context.workspace / "composer.json"
    if os.path.isfile(composer_json_path):
        return {
            "result": gcp.OPT_IN,
            "description": "Composer file found",
            "files": ["composer.json"]
        }
    
    has_php_files = any(context.workspace.glob("*.php"))
    if has_php_files:
        return {
            "result": gcp.OPT_IN,
            "description": ".php files found"
        }
    
    return {
        "result": gcp.OPT_OUT,
        "description": "No composer.json or .php files found"
    }

def build_fn(context: gcp.Context) -> None:
    """Build PHP runtime layer."""
    version = php_packaging.extract_version(context)
    if not version:
        version = "8.4.x"

    try:
        phpl = context.create_layer("php", ["build", "cache", "launch"])
    except Exception as e:
        raise RuntimeError(f"Error creating layer: {e}") from e

    installable_runtime = php_packaging.get_installable_runtime(context)
    
    try:
        runtime.install_tarball_if_not_cached(
            context,
            installable_runtime,
            version,
            phpl
        )
    except Exception as e:
        raise RuntimeError(f"Error installing PHP runtime: {e}") from e

    set_pecl_config(phpl)
    set_php_fpm_config(phpl)
    add_php_ini(context, phpl)

def set_pecl_config(layer: libcnb.Layer) -> None:
    """Set PECL configuration for the layer."""
    bin_path = layer.path / "bin" / "php"
    pear_install_dir = layer.path / "lib" / "php"
    
    layer.shared_environment.setdefault("PHP_PEAR_PHP_BIN", str(bin_path))
    layer.shared_environment.setdefault("PHP_PEAR_INSTALL_DIR", str(pear_install_dir))

def set_php_fpm_config(layer: libcnb.Layer) -> None:
    """Set PHP FPM configuration for the layer."""
    sbin_path = layer.path / "sbin"
    layer.launch_environment.append_to_path(str(sbin_path))

def add_php_ini(context: gcp.Context, layer: libcnb.Layer) -> None:
    """Add php.ini file to the layer."""
    etc_dir = layer.path / "etc"
    ini_file = etc_dir / PHP_INI_NAME
    
    try:
        etc_dir.mkdir(parents=True, exist_ok=True)
        with open(ini_file, 'w') as f:
            f.write(php_packaging.PHP_INI)
    except Exception as e:
        raise RuntimeError(f"Error creating php.ini: {e}") from e

    layer.launch_environment.setdefault("PHPRC", str(etc_dir))
