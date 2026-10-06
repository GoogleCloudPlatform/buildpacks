#!/bin/bash
# Copyright 2023 Google LLC
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

# The create-buildpackage.sh script creates a meta buildpack from source as a
# ".cnb" file using the `pack buildpack package` command.
#
# Note: This script should not be invoked directly; it is used in defs.bzl.
#
# Usage:
# ./create-buildpackage.sh <source tar> <cnb file name>

set -euox pipefail

readonly tar="${1:?tar path missing}"
readonly name="${2:?buildpackage name missing}"

# Blaze does not set $HOME which is required by pack.
readonly temp="$(mktemp -d)"
trap "rm -rf $temp" EXIT
export HOME="$temp/home"
mkdir "$HOME"

echo "Extracting tar:"
tar xvf "$tar" -C "$temp"

echo "Creating buildpackage:"
pack buildpack package "$name" --config="${temp}/package.toml" --pull-policy=never --format=file
