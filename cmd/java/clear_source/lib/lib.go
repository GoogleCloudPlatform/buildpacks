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
Implements java/clear_source buildpack library functions.
"""

import os
from typing import Optional

def detect_fn(context: dict) -> tuple[Optional[dict], Optional[str]]:
    """
    Detects if clear source should be applied.
    
    Args:
        context: The build context containing file information
        
    Returns:
        A tuple of (detect_result, error)
    """
    from ..pkg import clearsource
    result, err = clearsource.detect_fn(context)
    if result is not None or err is not None:
        return result, err
    
    files_to_check = ["pom.xml", "build.gradle", "build.gradle.kts"]
    for file in files_to_check:
        if os.path.exists(os.path.join(context.get('working_dir', ''), file)):
            return {'file_found': file}, None
            
    message = f"none of {', '.join(files_to_check)} found. Clearing source only supported on maven and gradle projects."
    return None, message

def build_fn(context: dict) -> Optional[str]:
    """
    Performs the clear source operation.
    
    Args:
        context: The build context containing file information
        
    Returns:
        An error if any occurred
    """
    from ..pkg import clearsource
    return clearsource.build_fn(context, ["target", "build"])
