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
Implements dart/pub buildpack.
The pub buildpack installs application dependencies using the pub package manager.
"""

import os
from typing import Dict, Optional

from buildpacks.gcpbuildpack import Context, DetectResult, OptInFileFound, OptOutFileNotFound
import buildpacks.dart as dart_pkg

pub_layer = "pub"
pub_cache_env = "PUB_CACHE"

def detect(ctx: Context) -> tuple[Optional[DetectResult], Exception]:
    try:
        pubspec_exists = ctx.file_exists("pubspec.yaml")
    except Exception as e:
        return (None, e)
    
    if not pubspec_exists:
        return (OptOutFileNotFound("pubspec.yaml"), None)
    
    return (OptInFileFound("pubspec.yaml"), None)

def build(ctx: Context) -> Optional[Exception]:
    try:
        layer = ctx.create_layer(pub_layer, ["build", "cache"])
    except Exception as e:
        return f"Error creating {pub_layer} layer: {e}"
    
    try:
        os.environ[pub_cache_env] = layer.path
    except Exception as e:
        return f"Error setting environment variable {pub_cache_env}: {e}"
    
    # Check if it's a Flutter project
    is_flutter, err = dart_pkg.is_flutter(ctx.application_root)
    if err is not None:
        return err
    
    command = ["flutter", "pub", "get"] if is_flutter else ["dart", "pub", "get"]
    
    try:
        ctx.exec(command, user_attribution=True)
    except Exception as e:
        return f"Error executing pub get: {e}"
    
    return None
