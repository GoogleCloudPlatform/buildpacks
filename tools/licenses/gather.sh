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

# The gather.sh script gathers license information for all dependencies.
#
# The script creates YAML files in the specified directory containing license
# information for all project dependencies as well as for all lifecycle version
# referenced in any builder.toml files under the `builder` directory. It also
# saves all licenses in the output directory.
#
# The output structure will be:
#   <directory>/
#     files/github.com/foo/bar/LICENSE
#     files/github.com/bar/baz/NOTICE
#     buildpacks.yaml
#     lifecycle-v0.7.4.yaml
#     lifecycle-v0.7.5.yaml
#
# Usage:
#   gather.sh <directory>

set -euo pipefail
shopt -s globstar  # Required for **.

DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_DIR="$(cd "${DIR}/../.." && pwd)"
LICENSE_DIR="${1:-}"

# Provides to_yaml.
source "$DIR/yaml.sh"

if [[ -z "${LICENSE_DIR}" ]]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

mkdir -p "${LICENSE_DIR}"
# Get absolute path to the output directory.
LICENSE_DIR="$(cd "$LICENSE_DIR" && pwd)"
LICENSE_FILES_DIR="${LICENSE_DIR}/files"
mkdir -p "${LICENSE_FILES_DIR}"

# Download go-licenses if not already installed.
if ! type -P go-licenses; then
  echo "Installing go-licenses"
  bin="$(mktemp -d)"
  GOBIN="$bin" go install github.com/google/go-licenses
  PATH="$bin:$PATH"
fi

echo "Gathering licenses for buildpacks"
go-licenses check "$PROJECT_DIR/..."
go-licenses save "$PROJECT_DIR/..."  --force --save_path "$LICENSE_FILES_DIR"
go-licenses csv "$PROJECT_DIR/..." | sort | to_yaml "buildpacks" "$LICENSE_FILES_DIR" > "${LICENSE_DIR}/buildpacks.yaml"

echo "Gathering licenses for lifecycle"
LIFECYCLE_DIR="$(mktemp -d)"

# Find all versions of lifecycle refereced in builder.toml files.
versions="$(grep "version = " builders/**/builder.toml | sed -r 's|.*"(.*)".*|\1|' | sort | uniq)"
if [[ -z "$versions" ]]; then
  echo "No lifecycle versions found in builder.toml files"
  exit 1
fi

for version in $versions; do
  echo
  echo "Downloading lifecycle-v${version}"
  curl -fsSL "https://github.com/buildpacks/lifecycle/archive/v${version}.tar.gz" | tar xz -C "$LIFECYCLE_DIR"

  echo "Gathering licenses for lifecycle-v${version}"
  echo "Note: it is expected to see 'error discovering URL' messages for official Go modules"
  echo

  lifecycle_dir="$LIFECYCLE_DIR/lifecycle-${version}"
  license_files_dir="$LICENSE_FILES_DIR/lifecycle-v${version}"
  # go-licenses only works from the package root.
  pushd "$lifecycle_dir"
  go-licenses check "$lifecycle_dir/..."
  go-licenses save "$lifecycle_dir/..." --force --save_path "$license_files_dir"
  go-licenses csv "$lifecycle_dir/..." | sort | to_yaml "lifecycle-${version}" "$license_files_dir" > "${LICENSE_DIR}/lifecycle-v${version}.yaml"
  popd
  # Add lifecycle license files to the other license files.
  rsync -a "$license_files_dir/" "$LICENSE_FILES_DIR/"
  rm -rf "$license_files_dir"
done


