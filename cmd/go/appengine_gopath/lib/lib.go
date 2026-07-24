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
Implements go/appengine_gopath buildpack.
The appengine_gopath buildpack sets $GOPATH and moves all gopath dependencies from _gopath/src/* to $GOPATH/src/*. The _gopath directory is created by go-app-stager during deployment.
It then checks for _gopath/main-package-path which exists if the user's main package was originally on $GOPATH/src locally.
If this file exists, the buildpack moves the main package to $GOPATH/src and sets the path to build $GOPATH/src/<path-to-main-package> where <path-to-main-package> is read from _gopath/main-package-path.
If this file doesn't exist, the buildpack sets the path to build to "./..." and removes the _gopath directory because the build will fail if there's more than one go package in application root.
"""

import os
import shutil
import subprocess
from pathlib import Path

class Context:
    def __init__(self, application_root):
        self.application_root = application_root

    def file_exists(self, path):
        return Path(os.path.join(self.application_root, path)).exists()

    def has_at_least_one(self, pattern):
        # This is a simplified version; actual implementation may vary
        for file in Path(self.application_root).rglob(pattern):
            return True
        return False

    def layer(self, name, build_layer=True):
        # Placeholder for layer creation logic
        pass

def detect_fn(ctx: Context) -> (bool, str):
    if not is_gae() and not is_flex():
        return False, "not a GAE Standard or Flex app."
    
    go_mod_exists = ctx.file_exists("go.mod")
    if go_mod_exists:
        return False, "go.mod found"
    
    has_go_files = ctx.has_at_least_one("*.go")
    if not has_go_files:
        return False, "no .go files found"
    
    return True, "go.mod file not found, assuming GOPATH build"

def build_fn(ctx: Context) -> None:
    # Implement the build logic here
    pass

def is_gae():
    # Check environment variables or other indicators for GAE
    return os.getenv("X_GOOGLE_TARGET_PLATFORM") == "gae"

def is_flex():
    # Check environment variables or other indicators for Flex
    return os.getenv("FLEX_ENV") == "flex"

if __name__ == "__main__":
    ctx = Context(os.getcwd())
    result, message = detect_fn(ctx)
    print(f"Detection result: {result}, Message: {message}")
