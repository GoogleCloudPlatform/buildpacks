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

"""Implements python/functions_framework buildpack."""

import os
import sys
from pathlib import Path

def detect():
    """Detect if the buildpack should be used."""
    function_target = os.getenv("FUNCTION_TARGET")
    
    if function_target:
        if is_pyproject_enabled():
            return True, None
        
        return True, [python.RequirementsProvidesPlan]
        
    return False, None

def build():
    """Build phase of the buildpack."""
    validate_source()
    
    # Check for syntax errors
    try:
        subprocess.run(["python3", "-m", "compileall", "-f", "-q", "."], check=True)
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Syntax validation failed: {e}") from e

    has_framework_dependency = is_framework_dependency_present()
    
    if not has_framework_dependency:
        if is_pyproject_enabled():
            raise ValueError("This project is using pyproject.toml but you have not included the Functions Framework in your dependencies. Please add it to your pyproject.toml.")
            
        if os.getenv(python.VendorPipDepsEnv):
            raise ValueError("Vendored dependencies detected, please add functions-framework to requirements.txt and download it using pip")
            
        add_framework_requirements()
        
    set_function_env_vars()
    add_web_process()

def validate_source():
    """Validate source files exist."""
    function_source = os.getenv(env.FunctionSource)
    
    if not function_source:
        main_py_exists = Path("main.py").exists()
        if not main_py_exists:
            raise ValueError(f"missing main.py and {env.FunctionSource} not specified. Either create the function in main.py or specify {env.FunctionSource}")
    else:
        source_path = Path(function_source)
        if not source_path.exists():
            raise ValueError(f"{env.FunctionSource} specified file {function_source!r} but it does not exist")

def is_framework_dependency_present():
    """Check if functions-framework is a dependency."""
    # Implement logic to check requirements.txt or pyproject.toml
    return False  # Placeholder implementation

def add_framework_requirements():
    """Add functions-framework to requirements."""
    requirements_path = Path("requirements.txt")
    with open(requirements_path, "a") as f:
        f.write("\nfunctions-framework\n")

def set_function_env_vars():
    """Set environment variables for the function."""
    pass  # Placeholder implementation

def add_web_process():
    """Add web process configuration."""
    pass  # Placeholder implementation

def is_pyproject_enabled():
    """Check if pyproject.toml is used."""
    return Path("pyproject.toml").exists()

# Constants
layer_name = "functions-framework"
function_framework = "functions-framework"
requirements_txt = "requirements.txt"
pyproject_toml = "pyproject.toml"
