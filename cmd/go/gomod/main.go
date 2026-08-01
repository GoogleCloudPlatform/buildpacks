# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Implements go/gomod buildpack.
The gomod buildpack downloads modules specified in go.mod.
"""

import os
from typing import Optional

def detect_fn(ctx: dict) -> tuple[Optional[dict], str]:
    """
    Detects if the application requires the gomod buildpack by checking for a go.mod file.

    Args:
        ctx: The context dictionary containing environment information.

    Returns:
        A tuple where the first element is either None or a dictionary indicating
        the detect result, and the second element is an error message.
    """
    go_mod_path = os.path.join(ctx["application_root"], "go.mod")
    
    if os.path.exists(go_mod_path):
        return {"file_found": "go.mod"}, ""
    else:
        return None, ""

def build_fn(ctx: dict) -> str:
    """
    Builds the application by downloading and verifying dependencies using go mod.

    Args:
        ctx: The context dictionary containing environment information.

    Returns:
        An error message if any issues occurred during the build.
    """
    # Create a temporary directory for GOPATH
    gopath_layer = os.path.join(ctx["layers_root"], "gopath")
    
    try:
        # Check for vendor directory
        vendor_path = os.path.join(ctx["application_root"], "vendor")
        
        if os.path.exists(vendor_path):
            print("Not downloading modules because there's a `vendor` directory.")
            return ""
            
        # Ensure go.mod is writable
        go_mod_path = os.path.join(ctx["application_root"], "go.mod")
        
        if not os.access(go_mod_path, os.W_OK):
            return "go.mod exists but is not writable"
            
        # Check for go.sum and generate it if missing
        go_sum_path = os.path.join(ctx["application_root"], "go.sum")
        
        if not os.path.exists(go_sum_path):
            print("Generating go.sum using 'go mod tidy'")
            result = run_command(["go", "mod", "tidy"], ctx)
            
            if result["exit_code"] != 0:
                return f"Running go mod tidy failed: {result['stderr']}"
                
        # Download modules
        print("Downloading modules...")
        result = run_command(["go", "mod", "download"], ctx)
        
        if result["exit_code"] != 0:
            return f"Running go mod download failed: {result['stderr']}"
            
        return ""
    except Exception as e:
        return str(e)

def run_command(command: list[str], ctx: dict) -> dict:
    """
    Runs a command with the specified environment context.

    Args:
        command: The command to execute.
        ctx: The context dictionary containing environment information.

    Returns:
        A dictionary containing the execution result.
    """
    env = os.environ.copy()
    env["GOPATH"] = gopath_layer
    env["GO111MODULE"] = "on"
    
    result = {}
    
    try:
        process = subprocess.run(
            command,
            cwd=ctx["application_root"],
            env=env,
            capture_output=True,
            text=True,
            check=False
        )
        
        result["exit_code"] = process.returncode
        result["stdout"] = process.stdout.strip()
        result["stderr"] = process.stderr.strip()
    except Exception as e:
        result["exit_code"] = 1
        result["stdout"] = ""
        result["stderr"] = str(e)
        
    return result
