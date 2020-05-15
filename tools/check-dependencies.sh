#!/bin/bash
# Copyright 2020 Google LLC
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

# The check-dependenceis.sh script checks that all required tools are installed.
#
# Note: This script is meant to be invoked using Blaze/Bazel.
#
# Usage:
#   ./check-dependencies.sh

set -euo pipefail

# Check that the command in $1 is in $PATH. If not, show what $PATH is and also
# the installation link in $2.
function check_in_path() {
  if ! type -P "$1"; then
    echo "$1 not found, please follow $2 and ensure that '$1' is in \$PATH:" >&2
    echo "  PATH=$PATH" >&2
    exit 1
  fi
}

echo "Checking that all required tools have been installed:"
check_in_path pack "https://buildpacks.io/docs/install-pack/"
check_in_path docker "https://docs.docker.com/install/"
check_in_path container-structure-test "https://github.com/GoogleContainerTools/container-structure-test#installation"
