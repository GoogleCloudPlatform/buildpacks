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
Implements php/composer-install buildpack.
The composer-install buildpack installs the composer dependency manager.
"""

import os
import sys
from pathlib import Path

import hashlib
import subprocess
import tempfile

import gcpbuildpack as gcp
from gcpbuildpack import (
    DetectResult,
    BuildContext,
    CacheLayer,
    BuildLayer,
    CacheMiss,
    CacheHit,
    InternalError,
)
from php import ComposerInstaller, ComposerInstallerCapability


composer_layer = "composer"
composer_json = "composer.json"
composer_setup = "composer-setup"
composer_ver = "2.2.24"
version_key = "version"
composer_sig_url = "https://composer.github.io/installer.sig"
composer_setup_url = "https://getcomposer.org/installer"


def detect(context: BuildContext) -> DetectResult:
    if os.getenv(env.FunctionTarget):
        # functions-frameworks buildpack expect composer sdk to be installed always.
        return gcp.OptInAlways()
    
    if not context.file_exists(composer_json):
        return gcp.OptOutFileNotFound(composer_json)
    
    return gcp.OptInFileFound(composer_json)


def build(context: BuildContext) -> None:
    # Get environment variables
    composer_version = os.getenv(php.ComposerVersion, composer_ver)

    # Create layer
    try:
        layer = context.create_layer(composer_layer, BuildLayer | CacheLayer)
    except Exception as e:
        raise RuntimeError(f"Creating {composer_layer} layer failed: {e}") from e

    # Check capability
    capability = context.get_capability(ComposerInstallerCapability)
    if capability is not None:
        if isinstance(capability, ComposerInstaller):
            capability.install(context, layer, composer_version)
            return
        else:
            raise InternalError(f"Capability {ComposerInstallerCapability} must implement ComposerInstaller")

    # Check cache
    meta_version = context.get_metadata(layer, version_key)
    if meta_version == composer_version:
        context.cache_hit(composer_layer)
        print("composer binary cache hit, skipping installation.")
        return
    
    context.cache_miss(composer_layer)
    
    try:
        context.clear_layer(layer)
    except Exception as e:
        raise RuntimeError(f"Clearing layer {layer.name} failed: {e}") from e

    # Download installer
    with tempfile.NamedTemporaryFile(suffix=f"-{composer_setup}.php", dir=layer.path) as installer:
        try:
            fetch_url(composer_setup_url, installer)
        except Exception as e:
            raise RuntimeError(f"Failed to download composer installer from {composer_setup_url}: {e}") from e

        # Verify checksum
        expected_sha = fetch_url(composer_sig_url).decode().strip()
        
        # Calculate actual SHA384
        with open(installer.name, 'rb') as f:
            actual_sha = hashlib.sha384(f.read()).hexdigest()

        if actual_sha != expected_sha:
            raise RuntimeError(
                f"Invalid composer installer found at {composer_setup_url}: "
                f"checksum for composer installer ({actual_sha}) does not match expected checksum of ({expected_sha})"
            )

        # Install Composer
        print(f"Installing Composer v{composer_version}")
        
        bin_dir = layer.path / "bin"
        bin_dir.mkdir(parents=True, exist_ok=True)
        
        install_cmd = f"php {installer.name} --install-dir {bin_dir} --filename composer --version {composer_version}"
        try:
            subprocess.run(["bash", "-c", install_cmd], check=True)
        except subprocess.CalledProcessError as e:
            raise RuntimeError(f"Failed to install Composer: {e}") from e

    # Update metadata
    context.set_metadata(layer, version_key, composer_version)


def fetch_url(url: str, output_path: Path) -> None:
    import urllib.request
    
    try:
        with urllib.request.urlopen(url) as response:
            with open(output_path, 'wb') as f:
                f.write(response.read())
    except Exception as e:
        raise RuntimeError(f"Failed to fetch {url}: {e}") from e
