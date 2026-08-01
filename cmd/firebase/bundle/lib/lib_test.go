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
Detect function implementation for Firebase bundle buildpack.
"""

import os

def detect(context):
    """
    Detects if it is a firebase apphosting application.
    
    Args:
        context: The build context containing environment information.
        
    Returns:
        A tuple indicating whether to opt-in or out, along with an error.
    """
    # This buildpack handles some necessary setup for future app hosting processes,
    # it should always run for any app hosting initial build.
    if not os.getenv("X_GOOGLE_TARGET_PLATFORM") == "fah":
        return gcp.OptOut("not a firebase apphosting application"), None
    
    # The environment variable is converted to a string "true" not true.
    use_generic = os.getenv(env.GOOGLE_USE_GENERIC_FIREBASEBUNDLE, "")
    if use_generic.lower() != "true":
        return gcp.OptOut("not using google.firebase.firebasebundle because GOOGLE_USE_GENERIC_FIREBASEBUNDLE is not true"), None
    
    return gcp.OptIn("firebase apphosting application"), None
