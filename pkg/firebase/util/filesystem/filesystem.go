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

import os
import re
from pathlib import Path

from firebase.error import faherror

app_hosting_yaml_pattern = re.compile(r'^apphosting(\.[a-z0-9_]+)?\.yaml$')

def detect_app_hosting_yaml_path(workspace_path: str, backend_root_directory: str) -> str:
    """
    Returns the absolute path to the apphosting.yaml file, or raises an error if not found.
    
    Args:
        workspace_path (str): The base directory of the workspace.
        backend_root_directory (str): The relative path to the backend root directory.

    Raises:
        faherror.InvalidRootDirectoryError: If the backend root directory doesn't exist.

    Returns:
        str: Absolute path to apphosting.yaml file.
    """
    backend_root = Path(workspace_path) / backend_root_directory
    if not backend_root.exists():
        raise faherror.InvalidRootDirectoryError(
            f"Backend root directory '{backend_root}' does not exist"
        )

    app_hosting_root = _detect_app_hosting_yaml_root(backend_root)
    return str(app_hosting_root / "apphosting.yaml")

def _detect_app_hosting_yaml_root(root: Path) -> Path:
    """
    Finds the root directory containing apphosting.yaml by searching upwards.

    Args:
        root (Path): The starting directory to search from.

    Returns:
        Path: The directory containing apphosting.yaml, or raises FileNotFoundError if not found.
    """
    current = root.resolve()
    while True:
        if _app_hosting_yaml_exists_in_dir(current):
            return current
        
        parent = current.parent
        if parent == current:  # Reached the topmost directory
            break
        
        current = parent

    raise FileNotFoundError("No apphosting.yaml file found in any parent directories")

def _app_hosting_yaml_exists_in_dir(directory: Path) -> bool:
    """
    Checks if any file matching apphosting.yaml pattern exists in the given directory.

    Args:
        directory (Path): Directory to check for apphosting.yaml files.

    Returns:
        bool: True if a matching file is found, False otherwise.
    """
    try:
        for entry in os.listdir(directory):
            if app_hosting_yaml_pattern.match(entry):
                return True
        return False
    except OSError as e:
        raise faherror.FileSystemError(f"Failed to read directory '{directory}': {e}") from e
