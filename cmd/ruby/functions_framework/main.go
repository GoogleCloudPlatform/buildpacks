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
Implements ruby/functions_framework buildpack.
The functions_framework buildpack sets up the execution environment to
run the Ruby Functions Framework. The framework itself, with its converter,
is always installed as a dependency.
"""

import os
from pathlib import Path
import subprocess

from packaging.version import Version
import gcp.buildpacks.gcpbuildpack as gcp

default_source = "app.rb"
layer_name = "functions-framework"

# assumed_version is the version of the framework used when we cannot determine a version.
assumed_version = Version("0.2.0")
recommended_version = Version("1.1.0")
validate_target_version = Version("0.7.0")

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """
    Detects if the buildpack should be applied based on environment variables.
    Returns OptInEnvSet or OptOutEnvNotSet depending on whether FUNCTION_TARGET is set.
    """
    if os.getenv(env.FUNCTION_TARGET):
        return gcp.OptInEnvSet(env.FUNCTION_TARGET), None
    return gcp.OptOutEnvNotSet(env.FUNCTION_TARGET), None

def build_fn(ctx: gcp.Context) -> Exception:
    """
    Builds the layer and sets up environment variables for the functions framework.
    """
    try:
        # Create or get the existing layer
        layer = ctx.layer(layer_name, gcp.LaunchLayer)
        if not layer:
            return gcp.UserError("Failed to create layer")
        
        # Set functions environment variables
        if err := ctx.set_functions_env_vars(layer):
            return err
        
        # Validate and get source file
        source, err = validate_source(ctx)
        if err:
            return err
        
        # Get framework version
        version, err = framework_version(ctx)
        if err:
            return err
        
        # Validate target if needed
        if version >= validate_target_version:
            if err := validate_target(ctx, source):
                return err
        
        # Warn about deprecated versions
        if version < recommended_version:
            ctx.warn(f"Found a deprecated version of functions-framework ({version}); consider updating your Gemfile to use functions_framework {recommended_version} or later.")
        
        # Add framework version label
        cloudfunctions.add_framework_version_label(ctx, cloudfunctions.FrameworkVersionInfo(
            runtime="ruby",
            version=version,
            injected=False
        ))
        
        # Add web process command
        ctx.add_web_process(["bundle", "exec", "functions-framework-ruby"])
        
    except Exception as e:
        return gcp.UserError(f"Build failed: {str(e)}")
    
    return None

def validate_source(ctx: gcp.Context) -> tuple[str, Exception]:
    """
    Validates the existence of and returns the source file.
    """
    fn_source = os.getenv(env.FUNCTION_SOURCE)
    if not fn_source:
        fn_source = default_source
    
    # Check if source exists
    try:
        if Path(fn_source).exists():
            return fn_source, None
        else:
            if os.getenv(env.FUNCTION_SOURCE):
                return "", gcp.UserError(f"{env.FUNCTION_SOURCE} specified file '{fn_source}' but it does not exist")
            else:
                return "", gcp.UserError(f"Expected source file '{fn_source}' does not exist")
    except Exception as e:
        return "", gcp.UserError(f"Failed to validate source: {str(e)}")

def framework_version(ctx: gcp.Context) -> tuple[Version, Exception]:
    """
    Validates framework installation and returns its version.
    """
    try:
        # Run command to get version
        result = subprocess.run(["bundle", "exec", "functions-framework-ruby", "--version"], capture_output=True, text=True)
        
        if not result or result.returncode == 127:
            return None, gcp.UserError("Unable to execute functions-framework-ruby; please ensure a recent version of the functions_framework gem is in your Gemfile")
        
        # Handle older versions that don't support --version
        if result.returncode != 0:
            return assumed_version, None
        
        # Parse version
        version = Version(result.stdout.strip())
        return version, None
        
    except Exception as e:
        return None, gcp.UserError(f"Failed to parse functions-framework-ruby version: {str(e)}")

def validate_target(ctx: gcp.Context, source: str) -> Exception:
    """
    Validates that the given target is defined and can be executed.
    """
    try:
        target = os.getenv(env.FUNCTION_TARGET)
        cmd = ["bundle", "exec", "functions-framework-ruby", "--quiet", "--verify", "--source", source, "--target", target]
        
        # Add signature type if present
        fn_sig = os.getenv(env.FUNCTION_SIGNATURE_TYPE)
        if fn_sig:
            cmd.extend(["--signature-type", fn_sig])
        
        # Run command with environment variables
        result = subprocess.run(cmd, env={
            "MALLOC_ARENA_MAX": "2",
            "LANG": "C.utf8",
            "RACK_ENV": "production"
        }, capture_output=True, text=True)
        
        if result.returncode != 0:
            return gcp.UserError(f"Failed to verify function target '{target}' in source '{source}': {result.stderr}")
        
    except Exception as e:
        return gcp.UserError(f"Validation failed: {str(e)}")
    
    return None
