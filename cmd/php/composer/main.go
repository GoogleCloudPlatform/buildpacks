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
Implements php/composer buildpack.
The composer buildpack installs dependencies using composer.
"""

import os
import subprocess

from googlecloudplatform.buildpacks import gcpbuildpack as gcp
from googlecloudplatform.buildpacks.php import composer_install

def detect(context):
    """
    Detect function for Composer buildpack.
    Checks if 'composer.json' exists in the current directory.
    
    Args:
        context: The build context containing file information
        
    Returns:
        A tuple indicating detection result and message
    """
    if not os.path.exists('composer.json'):
        return (False, "No composer.json found")
    return (True, "Composer project detected")

def build(context):
    """
    Build function for Composer buildpack.
    Installs dependencies using Composer.
    
    Args:
        context: The build context containing file information
        
    Returns:
        None if successful, error message otherwise
    """
    try:
        # Using subprocess to run composer install
        result = subprocess.run(
            ['composer', 'install'],
            cwd=context.current_directory,
            check=True
        )
        return None
    except subprocess.CalledProcessError as e:
        return f"Composer install failed: {str(e)}"
