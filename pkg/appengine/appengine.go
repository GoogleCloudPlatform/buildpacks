# Copyright 2020 Google LLC
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
Package appengine contains buildpack library code for all runtimes.
"""

import os
from typing import Optional

import appstart  # Assuming this is a Python package with similar functionality
import env  # Assuming this is a Python package with environment constants
import gcpbuildpack as gcp  # Assuming this is the Python package


DefaultCommand = "/serve"
DepWarning = "There is a dependency on App Engine APIs, but they are not enabled in your app.yaml. Set the app_engine_apis property."
IndirectDepWarning = "There is an indirect dependency on App Engine APIs, but they are not enabled in your app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property."
UnusedAPIWarning = "App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app.yaml."

def getEntrypoint(ctx: gcp.Context, eg: Optional[appstart.EntrypointGenerator]) -> tuple[appstart.Entrypoint, str]:
    """
    Retrieves the entrypoint configuration.
    
    Args:
        ctx: The build context
        eg: An optional entrypoint generator
    
    Returns:
        A tuple containing the entrypoint and an error message (if any)
    """
    if os.getenv(env.Entrypoint):
        return appstart.Entrypoint(
            type=appstart.EntrypointUser,
            command=os.getenv(env.Entrypoint)
        ), None

    if eg:
        return eg(ctx), None

    return appstart.Entrypoint(
        type=appstart.EntrypointDefault,
        command=DefaultCommand
    ), None

def getConfig(ctx: gcp.Context, runtime: str, eg: Optional[appstart.EntrypointGenerator]) -> tuple[appstart.Config, str]:
    """
    Builds the application configuration.
    
    Args:
        ctx: The build context
        runtime: The target runtime
        eg: An optional entrypoint generator
    
    Returns:
        A tuple containing the configuration and an error message (if any)
    """
    config = appstart.Config()

    # Runtime handling
    if os.getenv(env.Runtime):
        ctx.log(f"Using {env.Runtime}: {os.getenv(env.Runtime)}")
        config.runtime = os.getenv(env.Runtime)
    else:
        ctx.debug(f"Using runtime: {runtime}")
        config.runtime = runtime

    # Entrypoint handling
    ep, err = getEntrypoint(ctx, eg)
    if err:
        return None, f"getting entrypoint: {err}"
    config.entrypoint = ep

    # Main executable handling
    if os.getenv(env.GAEMain):
        ctx.log(f"Using {env.GAEMain}: {os.getenv(env.GAEMain)}")
        config.main_executable = os.getenv(env.GAEMain)

    ctx.log(f"Using config: {config}")
    return config, None

def Build(ctx: gcp.Context, runtime: str, eg: Optional[appstart.EntrypointGenerator]) -> str:
    """
    Builds the application using the specified configuration.
    
    Args:
        ctx: The build context
        runtime: The target runtime
        eg: An optional entrypoint generator
    
    Returns:
        An error message if any occurs during building
    """
    config, err = getConfig(ctx, runtime, eg)
    if err:
        return f"building config: {err}"

    if config.write(ctx) != "":
        return "Error writing configuration"

    ctx.add_web_process(["/start"])
    return ""

def ApisEnabled(ctx: gcp.Context) -> tuple[bool, str]:
    """
    Checks if App Engine APIs are enabled.
    
    Args:
        ctx: The build context
    
    Returns:
        A tuple containing a boolean indicating if APIs are enabled and an error message (if any)
    """
    val = os.getenv(env.AppEngineAPIs)
    if not val:
        return False, None

    try:
        return bool(val), None
    except ValueError as e:
        return False, f"parsing {val} from {env.AppEngineAPIs}: {e}"

def OptInTargetPlatformGAE() -> gcp.DetectResult:
    """
    Returns a DetectResult for opting in when target platform is App Engine.
    """
    return gcp.OptInEnvSet(env.TargetPlatformAppEngine)

def OptOutTargetPlatformNotGAE() -> gcp.DetectResult:
    """
    Returns a DetectResult for opting out when target platform is not App Engine.
    """
    return gcp.OptOut(f"{env.XGoogleTargetPlatform} not set to {env.TargetPlatformAppEngine}")
