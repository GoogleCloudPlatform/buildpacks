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
#     buildpacks/licenses.yaml
#     lifecycle-v0.7.4/licenses.yaml
#     lifecycle-v0.7.5/licenses.yaml
#     files/buildpacks/...
#     files/lifecycle-v0.7.4/...
#     files/lifecycle-v0.7.5/...
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
  echo
fi

# licenses generates licenses.yaml and licenses files for the given component
# whose source code is in the specified directory.
function licenses() {
  local readonly component="${1?}"   # component name
  local readonly image_dir="${2?}"   # name of directory with licenses on image
  local readonly source_dir="${3?}"  # path to source

  echo "Note: it is expected to see 'error discovering URL' messages"

  mkdir -p "$LICENSE_DIR/$component"
  pushd "$source_dir"
  go-licenses check "./..."
  go-licenses save "./..."  --force --save_path "$LICENSE_FILES_DIR/$component"
  go-licenses csv "./..." | to_yaml "$image_dir" "$LICENSE_FILES_DIR/$component" > "$LICENSE_DIR/$component/licenses.yaml"
  popd
}


echo "Gathering licenses for buildpacks"
licenses "buildpacks" "buildpacks" "$PROJECT_DIR"

echo "Gathering licenses for lifecycle"
LIFECYCLE_DIR="$(mktemp -d)"
trap "rm -rf $LIFECYCLE_DIR" EXIT

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
  licenses "lifecycle-v${version}" "lifecycle" "$LIFECYCLE_DIR/lifecycle-${version}"
done
