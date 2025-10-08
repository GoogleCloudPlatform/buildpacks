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

# This awk script looks for the [buildpack] section,
# and then extracts id and version from within that section, without
# depending on their order. It stops processing when it hits the next section.
readonly awk_script='
  BEGIN { in_buildpack = 0; id = ""; version = "" }
  /\[buildpack\]/ { in_buildpack = 1; next }
  /\[.*\]/ { if (in_buildpack) { exit } }
  in_buildpack && /^\s*id\s*=\s*".*"/ {
    match($0, /"(.*)"/);
    id = substr($0, RSTART + 1, RLENGTH - 2);
  }
  in_buildpack && /^\s*version\s*=\s*".*"/ {
    match($0, /"(.*)"/);
    version = substr($0, RSTART + 1, RLENGTH - 2);
  }
  END {
    if (id != "" && version != "") {
      print id "@" version
    }
  }
'

buildpacks=()

# Enable recursive globbing and prevent errors if no files are found.
shopt -s globstar nullglob

# Find buildpack archives and extract info from their buildpack.toml
for bp_archive in "${search_dir}"/**/*.{tgz,tar,cnb}; do
  if [[ ! -f "$bp_archive" ]]; then
    continue
  fi
  # Use `|| true` to prevent exiting if tar fails or grep finds no match.
  toml_path=$(tar -tf "${bp_archive}" 2>/dev/null | grep 'buildpack.toml$' | head -n 1 || true)
  if [[ -n "$toml_path" ]]; then
    # Use `|| true` here as well in case awk fails.
    buildpack_info=$(tar -xOf "${bp_archive}" "${toml_path}" 2>/dev/null | awk "$awk_script" || true)
    if [[ -n "$buildpack_info" ]]; then
      buildpacks+=("$buildpack_info")
    fi
  fi
done

# Also find standalone buildpack.toml files, for/if buildpacks that are not archived.
for toml_file in "${search_dir}"/**/buildpack.toml; do
  if [[ ! -f "$toml_file" ]]; then
    continue
  fi
  buildpack_info=$(awk "$awk_script" "$toml_file")
  if [[ -n "$buildpack_info" ]]; then
    buildpacks+=("$buildpack_info")
  fi
done

if [[ "${#buildpacks[@]}" -gt 0 ]]; then
  # There could be duplicates, so we remove them.
  unique_buildpacks=($(printf "%s\n" "${buildpacks[@]}" | sort -u))
  flatten_list=$(IFS=,; echo "${unique_buildpacks[*]}")
  # Final output goes to stdout for the calling script.
  echo "--flatten ${flatten_list}"
else
  # Final output goes to stdout for the calling script.
  echo ""
fi
