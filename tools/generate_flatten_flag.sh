#!/bin/bash
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

# The generate_flatten_flag.sh script generates the --flatten flag for pack builder create.
#
# Usage:
# ./generate_flatten_flag.sh <directory>
#
# It recursively finds all buildpack.toml files in the given directory,
# and also looks inside buildpack archives (*.tgz, *.tar, *.cnb) for buildpack.toml files.
# It then extracts the buildpack ID and version, and constructs the --flatten flag argument.

set -euo pipefail

if [[ "$#" -ne 1 ]]; then
    echo "Usage: $0 <directory>" >&2
    exit 1
fi

readonly search_dir="$1"

if [[ ! -d "$search_dir" ]]; then
  echo "Error: Directory '$search_dir' not found." >&2
  exit 1
fi

# This awk script looks for id and version in lines following [buildpack].
readonly awk_script='
  BEGIN { in_buildpack = 0; id = ""; version = "" }
  /\[buildpack\]/ { in_buildpack = 1; next }
  in_buildpack && /^\[/ { in_buildpack = 0 }
  in_buildpack && /id *= *".*"/ {
    val = $0;
    gsub(/^ *id *= *"/, "", val);
    gsub(/" *$/, "", val);
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", val);
    id = val;
  }
  in_buildpack && /version *= *".*"/ {
    val = $0;
    gsub(/^ *version *= *"/, "", val);
    gsub(/" *$/, "", val);
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", val);
    version = val;
  }
  END {
    if (id != "" && version != "") {
      print id "@" version
    }
  }
'

buildpacks=()

while IFS= read -r -d $'\0' bp_archive; do
  toml_path=$(tar -tf "${bp_archive}" 2>/dev/null | grep 'buildpack\.toml$' | head -n 1 || true)
  if [[ -n "$toml_path" ]]; then
    buildpack_info=$(tar -xOf "${bp_archive}" "${toml_path}" 2>/dev/null | awk "${awk_script}" || true)
    if [[ -n "$buildpack_info" ]]; then
      buildpacks+=("$buildpack_info")
    fi
  fi
done < <(find "$search_dir" -type f \( -name "*.tgz" -o -name "*.tar" -o -name "*.cnb" \) -print0)

while IFS= read -r -d $'\0' toml_file; do
  buildpack_info=$(awk "${awk_script}" "$toml_file")
  if [[ -n "$buildpack_info" ]]; then
    buildpacks+=("$buildpack_info")
  fi
done < <(find "$search_dir" -type f -name "buildpack.toml" -print0)

if [[ "${#buildpacks[@]}" -gt 0 ]]; then
  unique_buildpacks=($(printf "%s\n" "${buildpacks[@]}" | sort -u))
  flatten_list=$(IFS=,; echo "${unique_buildpacks[*]}")
  echo "--flatten ${flatten_list}"
else
  echo ""
fi
