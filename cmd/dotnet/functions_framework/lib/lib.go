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
Implements dotnet/functions_framework buildpack.
The functions_framework buildpack sets up the execution environment for functions.
"""

import os

def detect_fn(ctx):
    from googlecloudplatform.buildpacks.gcpbuildpack import OptInEnvSet, OptOutEnvNotSet
    if os.getenv(env.FUNCTION_TARGET) is not None:
        return OptInEnvSet(env.FUNCTION_TARGET), None
    return OptOutEnvNotSet(env.FUNCTION_TARGET), None

def build_fn(ctx):
    try:
        layer = ctx.layer(layer_name)
        if err := ctx.set_functions_env_vars(layer); err != nil {
            return fmt.Errorf("creating %v layer: %w", layerName, err)
        }
        return nil
    except Exception as e:
        raise RuntimeError(f"Error creating {layer_name} layer") from e

# Constants
layer_name = "functions-framework"
