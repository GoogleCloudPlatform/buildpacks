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
Implements the java/entrypoint buildpack.
"""

import os
from typing import Optional, Dict

MANIFEST_PATH = "META-INF/MANIFEST.MF"

def detect_fn(context: Dict) -> str:
    """
    Detects if the current environment should use this buildpack.
    
    Args:
        context (Dict): The context containing file information.
        
    Returns:
        str: Detection result indicating whether to opt-in or opt-out of the buildpack.
    """
    manifest_exists = os.path.exists(os.path.join(context.get("work_dir", ""), MANIFEST_PATH))
    
    if manifest_exists:
        return f"Opting in based on presence of {MANIFEST_PATH}"
    else:
        return f"Opting out as {MANIFEST_PATH} not found"

def build_fn(context: Dict) -> None:
    """
    Builds the application using the exploded jar.
    
    Args:
        context (Dict): The context containing necessary information for building.
    """
    # In a real implementation, this would include logic to extract and run the main class
    work_dir = context.get("work_dir", "")
    manifest_path = os.path.join(work_dir, MANIFEST_PATH)
    
    if not os.path.exists(manifest_path):
        raise FileNotFoundError(f"{MANIFEST_PATH} does not exist in {work_dir}")
        
    # Simulate extracting and running the main class
    print("Extracting Main-Class from MANIFEST...")
    print("Running application with extracted main class")
