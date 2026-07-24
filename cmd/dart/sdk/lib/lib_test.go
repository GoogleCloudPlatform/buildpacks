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
Implements dart/sdk buildpack.
The sdk buildpack installs the Dart SDK.
"""

import os
from pathlib import Path
import logging

const (
    DART_LAYER = "dart"
    DEFAULT_VERSION = "2.16.0"
    FLUTTER_LAYER = "flutter"
    DEFAULT_FLUTTER_VERSION = "3.29.3"
)

def detect(ctx):
    """
    Detects if the Dart SDK is needed for the build.
    
    Args:
        ctx: The build context
        
    Returns:
        A tuple of (detect_result, error)
    """
    # Check for pubspec.yaml or dart files
    pubspec_exists = os.path.exists("pubspec.yaml")
    if pubspec_exists:
        return gcp.OptInFileFound("pubspec.yaml"), None
    
    dart_files = Path(".").glob("*.dart")
    if list(dart_files):
        return gcp.OptIn("found .dart files"), None
    
    # If no dart files or pubspec.yaml found
    return gcp.OptOut("neither pubspec.yaml nor any .dart files found"), None

def build(ctx):
    """
    Builds the Dart SDK layer.
    
    Args:
        ctx: The build context
        
    Returns:
        error if any occurs
    """
    # Check if it's a Flutter project
    is_flutter, err = dart.is_flutter(ctx.application_root)
    if err is None and is_flutter:
        return build_flutter_fn(ctx)
    
    return build_dart_fn(ctx)

def build_dart_fn(ctx):
    """
    Builds the Dart SDK layer.
    
    Args:
        ctx: The build context
        
    Returns:
        error if any occurs
    """
    version, err = dart.detect_sdk_version()
    if err is not None:
        return err
    
    # Create or get the Dart layer
    drl, err = ctx.layer(DART_LAYER, gcp.BuildLayer | gcp.CacheLayer)
    if err is not None:
        return fmt.Errorf(f"creating {DART_LAYER} layer: {err}")
    
    # Check cache
    if runtime.is_cached(ctx, drl, version):
        ctx.cache_hit(DART_LAYER)
        logging.info("Runtime cache hit, skipping installation.")
        return
    
    # Install Dart SDK
    return runtime.install_dart_sdk(ctx, drl, version)

def build_flutter_fn(ctx):
    """
    Builds the Flutter SDK layer.
    
    Args:
        ctx: The build context
        
    Returns:
        error if any occurs
    """
    version, archive, err = dart.detect_flutter_sdk_archive()
    if err is not None:
        return err
    
    # Create or get the Flutter layer
    drl, err = ctx.layer(FLUTTER_LAYER, gcp.BuildLayer | gcp.CacheLayer)
    if err is not None:
        return fmt.Errorf(f"creating {FLUTTER_LAYER} layer: {err}")
    
    # Check cache
    if runtime.is_cached(ctx, drl, version):
        ctx.cache_hit(FLUTTER_LAYER)
        logging.info("Runtime cache hit, skipping installation.")
        return
    
    # Install Flutter SDK
    return runtime.install_flutter_sdk(ctx, drl, version, archive)
