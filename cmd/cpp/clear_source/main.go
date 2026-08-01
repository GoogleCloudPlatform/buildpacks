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
Implements cpp/clear_source buildpack.
The clear_source buildpack deletes source files after building the application.
"""

import os
from typing import Dict, Any, Optional

def detect_fn(context: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    """
    Detects if the clear_source buildpack should be applied.

    Args:
        context: The build context containing environment variables and other metadata.

    Returns:
        A dictionary indicating whether the buildpack should opt-in or None.
    """
    # Check for GOOGLE_CLEAR_SOURCE environment variable
    if os.environ.get("GOOGLE_CLEAR_SOURCE") == "true":
        return {"enabled": True}
    
    # Check if devmode is enabled (as in original Go code)
    if os.environ.get("GOOGLE_DEVMODE") == "true":
        return None
    
    return None

def build_fn(context: Dict[str, Any]) -> None:
    """
    Executes the clear_source buildpack logic.

    Args:
        context: The build context containing environment variables and other metadata.
    """
    # Note: In Go code, this calls into a clearsource package function,
    # but in Python we would need to implement similar functionality directly
    # or import from another module if available.
    pass
