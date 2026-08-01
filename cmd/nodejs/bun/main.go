# Copyright 2026 Google LLC
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
Package lib implements nodejs/bun buildpack.
The bun buildpack installs dependencies using bun package manager.
"""

import os
import subprocess
from pathlib import Path

class Context:
    def __init__(self, application_root):
        self.application_root = application_root
    
    def file_exists(self, path):
        return (Path(self.application_root) / path).exists()
    
    def layer(self, name, *args):
        # Implement layer management logic here
        pass
    
    def exec(self, command, **kwargs):
        return subprocess.run(command, check=True)

class DetectResult:
    def __init__(self, opt_in=False, opt_out=False, reason=None):
        self.opt_in = opt_in
        self.opt_out = opt_out
        self.reason = reason

def detect_fn():
    context = Context(os.getcwd())
    
    # Check for package.json
    if not context.file_exists("package.json"):
        return DetectResult(opt_out=True, reason="package.json not found")
    
    # Check for bun lock files
    if context.file_exists("bun.lockb") or context.file_exists("bun.lock"):
        return DetectResult(opt_in=True)
    
    # Check environment variable
    package_manager = os.environ.get("GOOGLE_PACKAGE_MANAGER", "")
    if package_manager == "bun":
        return DetectResult(opt_in=True, reason="GOOGLE_PACKAGE_MANAGER=bun")
    
    return DetectResult(opt_out=True, reason="No bun lock files found")

def build_fn():
    context = Context(os.getcwd())
    
    # Install dependencies
    try:
        _install_bun(context)
        _bun_install_modules(context)
        
        # Set up environment
        env_layer = context.layer("env")
        bin_path = os.path.join(context.application_root, "node_modules", ".bin")
        env_layer.set_path(bin_path)
        
        # Configure entrypoint
        if not is_dev_sync():
            add_web_process(["npm", "run", "start"])
    except Exception as e:
        print(f"Build error: {str(e)}", file=sys.stderr)
        raise

def _install_bun(context):
    # Implement bun installation logic here
    pass

def _bun_install_modules(context):
    # Implement bun install modules logic here
    pass

def is_dev_sync():
    return os.environ.get("DEVSYNC") == "true"

def add_web_process(command):
    print(f"Added web process: {command}")
