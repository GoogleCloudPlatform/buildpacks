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

set -euo pipefail

function license_header() {
  cat << EOF
# List of licenses and source for software packages included in image.
#
# Valid keys:
#   package: (required) Name of the package
#   license_name: (required) Name of the license, e.g., APL-2, MIT. "Custom" may be used for composite license files.
#   license_url: Direct link to license file in OSS project.
#   linking_binaries: Name of image binaries that use the software.
#   license_path: Absolute path to license file on the image.
#   source_path: Absolute path to the source on the image.
#
EOF
}

function license_entry() {
  local files="${1?}"
  local package="${2?}"
  local license_url="${3?}"
  local license_name="${4?}"
  # Do not emit license_url as go modules often do not have a resolvable URL.
  echo "- package: $package"
  echo "  license_name: $license_name"

  # Try to use the go-licenses-derived licenseUrl to find the license file
  # (note licenses_url = "Unknown" for golang.org modules). If not found, look
  # for well-known license files.
  local license_path="$(echo "${license_url}" | sed 's;https://\(.*\)/blob/master\(/.*\);\1\2;')"
  # Regex from https://github.com/google/go-licenses/blob/master/licenses/find.go#L26.
  local regex=".*/\(LICEN\(S\|C\)E\|COPYING\|README\|NOTICE\)\(\..+\)?$"
  local license_file="$(find "${files}/${package}" -regex "$regex" -printf '%P\n' | head -1)"
  if [ -f "${files}/${license_path}" ]; then
    echo "  license_path: /usr/local/share/licenses/buildpacks/${license_path}"
  elif [ -f "${files}/${package}/${license_file}" ]; then
    echo "  license_path: /usr/local/share/licenses/buildpacks/${package}/${license_file}"
  else
    echo "ERROR: unable to find license file for ${package} under ${files}/${package}" 1>&2
    exit 1
  fi
}

function to_yaml() {
  licenses_dir="${2?}"
  license_header

  # The sort flags make sorting consistent. The default behavior may differ
  # based on machine setup. For example, The Bazel Cloud Builder seems to sort
  # special characters and uppercase characters differently than local sort.
  cat - \
    | sort --ignore-case --stable --dictionary-order \
    | (IFS=","; while read package license_url license_name; do \
        echo; \
        license_entry $licenses_dir $package $license_url $license_name; \
      done)
}
