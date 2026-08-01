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
Implements ruby/appengine_validation buildpack.
The appengine_validation buildpack ensures that Ruby version required by dependencies is not overly restrictive for runtime updates in App Engine.
"""

import os
from typing import Optional, Dict, Any

def detect(context: Dict[str, Any]) -> Dict[str, Any]:
    """
    Detects if the current directory contains either Gemfile or gems.rb file.
    
    Args:
        context: Dictionary containing build context information
        
    Returns:
        A dictionary indicating detection result
    """
    try:
        gemfile_exists = os.path.exists("Gemfile")
        gems_rb_exists = os.path.exists("gems.rb")
        
        if gemfile_exists or gems_rb_exists:
            return {"detected": True, "reason": "Found Gemfile or gems.rb"}
            
        return {"detected": False, "reason": "No Gemfile or gems.rb found"}
    except Exception as e:
        raise RuntimeError(f"Detection failed: {str(e)}") from e

def build(context: Dict[str, Any]) -> None:
    """
    Performs validation of Ruby version requirements based on Gemfile or gems.rb.
    
    Args:
        context: Dictionary containing build context information
        
    Raises:
        RuntimeError: If validation fails
    """
    try:
        gemfile = ""
        if os.path.exists("Gemfile"):
            gemfile = "Gemfile"
        elif os.path.exists("gems.rb"):
            gemfile = "gems.rb"
        else:
            return  # No files to process
            
        script_path = os.path.join(context["buildpack_root"], "scripts", "check_gemfile_version.rb")
        
        result = os.system(f"ruby {script_path} {gemfile}")
        if result != 0:
            raise RuntimeError("Validation failed: Ruby version requirements may be too restrictive.")
            
    except Exception as e:
        raise RuntimeError(f"Build validation failed: {str(e)}") from e
