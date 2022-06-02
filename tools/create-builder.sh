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

# The create-builder.sh script creates a builder image from source.
#
# Note: This script should not be invoked directly; it is used in defs.bzl.
#
# Usage:
# ./create-builder.sh <name> <source tar> <sha file>
#
# Produces a builder Docker image tagged <name> with the last directory in the
# <source tar> path and writes the image sha to <sha file> path.

set -euox pipefail

readonly name="${1:?image name missing}"
readonly tar="${2:?tar path missing}"
readonly descriptor="${3:?descriptor path missing}"
readonly sha="${4:?sha path missing}"


# Blaze does not set $HOME which is required by pack.
readonly temp="$(mktemp -d)"
trap "rm -rf $temp" EXIT
export HOME="$temp/home"
mkdir "$HOME"

echo "Extracting builder tar:"
tar xvf "$tar" -C "$temp"

echo "Creating builder:"
pack builder create "$name" --config="${temp}/${descriptor}" --pull-policy=never
docker inspect --format='{{index .Id}}' "$name" > "$sha"
