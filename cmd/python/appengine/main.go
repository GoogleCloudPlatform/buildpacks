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

"""Implements python/appengine buildpack. The appengine buildpack sets the image entrypoint."""

import os
import re
import subprocess
from typing import Optional

import semver

import gcpbuildpack
import appengine
import appstart


_VERSION_REGEX = re.compile(r'(?m)^Version:\s+(.*)$')
_MIN_VERSION = semver.Version.parse("19.0.0")

def detect_fn(ctx: gcpbuildpack.Context) -> tuple[gcpbuildpack.DetectResult, Optional[str]]:
    """Detect function for the appengine buildpack."""
    if os.getenv('GAE_ENV'):
        return (gcpbuildpack.DetectResult.OPT_IN, "Deployment environment is GAE.")
    return (gcpbuildpack.DetectResult.OPT_OUT, "Deployment environment is not GAE.")

def build_fn(ctx: gcpbuildpack.Context) -> None:
    """Build function for the appengine buildpack."""
    validate_app_engine_apis(ctx)
    appengine.build(ctx, "python", entrypoint)

def validate_app_engine_apis(ctx: gcpbuildpack.Context) -> None:
    """Validate App Engine APIs are properly configured."""
    supports_apis = appengine.apisEnabled(ctx)
    using_app_engine = app_engine_in_deps(ctx)
    
    if supports_apis and not using_app_engine:
        ctx.warn(appengine.UNUSED_API_WARNING)
    if not supports_apis and using_app_engine:
        ctx.warn(appengine.DEP_WARNING)

def entrypoint(ctx: gcpbuildpack.Context) -> tuple[appstart.Entrypoint, Optional[str]]:
    """Determine the entrypoint for the application."""
    result = subprocess.run(
        ["python3", "-m", "pip", "show", "gunicorn"],
        capture_output=True,
        text=True
    )
    
    if result.returncode == 1:
        return (None, "gunicorn not installed: " + result.stdout)
    if result.returncode != 0:
        return (None, f"pip show gunicorn failed: {result.stderr}")
    
    match = _VERSION_REGEX.search(result.stdout)
    if not match or len(match.groups()) < 1:
        return (None, f"unable to find gunicorn version in output: {result.stdout}")
    
    version_str = match.group(1)
    try:
        version = semver.Version.parse(version_str)
    except ValueError as err:
        return (None, f"unable to parse gunicorn version string '{version_str}': {err}")
    
    if version < _MIN_VERSION:
        ctx.warn(f"Installed gunicorn version '{version}' is less than supported version '{_MIN_VERSION}'.")
    
    return (
        appstart.Entrypoint(
            type=appstart.EntrypointType.DEFAULT,
            command=appengine.DEFAULT_COMMAND
        ),
        None
    )

def app_engine_in_deps(ctx: gcpbuildpack.Context) -> bool:
    """Check if appengine-python-standard is installed."""
    result = subprocess.run(
        ["python3", "-m", "pip", "show", "appengine-python-standard"],
        capture_output=True,
        text=True
    )
    
    return result.returncode == 0
