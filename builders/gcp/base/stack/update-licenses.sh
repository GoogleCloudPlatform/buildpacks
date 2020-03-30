#!/usr/bin/env bash
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

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# TOPDIR is the top-level project directory
TOPDIR="$(cd "${DIR}/../../../.." && pwd)"

# Use github.com/google/go-licenses to extract third-party licenses for
# Go libraries used in this project to licenses/, and to build a summary
# licenses.yaml.

LICENSES_BASE="licenses"

# Extract Go licenses for third-party libraries used in the GCP Buildpacks
# and other binaries included in the builder images like the CNB Lifecycle.
# Use --force to cause the save_path to be deleted.
go run github.com/google/go-licenses save --force \
  --save_path "${DIR}/${LICENSES_BASE}" \
  "${TOPDIR}/..." \
  github.com/buildpacks/lifecycle

# Build licenses.yaml.
cat > licenses.yaml << EOF
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

outputLicenseEntry() {
  # do not emit the license_url as go modules often do not have a resolvable URL
  echo "- package: $package"
  echo "  license_name: $licenseName"

  # try to use the go-licenses-derived licenseUrl to find the license file
  # (note licenseUrl = "Unknown" for go modules).  If not found, look for
  # well-known license files.
  local license_path=$(echo "${licenseUrl}" | sed 's;https://\(.*\)/blob/master\(/.*\);\1\2;')
  if [ -f "${DIR}/${LICENSES_BASE}/${license_path}" ]; then
    echo "  license_path: /usr/share/doc/${license_path}"
  elif [ -f "${DIR}/${LICENSES_BASE}/${package}/LICENSE" ]; then
    echo "  license_path: /usr/share/doc/${package}/LICENSE"
  elif [ -f "${DIR}/${LICENSES_BASE}/${package}/COPYING" ]; then
    echo "  license_path: /usr/share/doc/${package}/COPYING"
  elif [ -f "${DIR}/${LICENSES_BASE}/${package}/NOTICE" ]; then
    echo "  license_path: /usr/share/doc/${package}/NOTICE"
  else
    echo "ERROR: unable to find license file for ${package}" 1>&2
    exit 1
  fi
}

# example output from `wgo-licenses csv`:
#   github.com/blang/semver,https://github.com/blang/semver/blob/master/LICENSE,MIT
echo "Note: it is expected to see 'error discovering URL' messages for go modules"
go run github.com/google/go-licenses csv \
  "${TOPDIR}/..." \
  github.com/buildpacks/lifecycle \
| sort \
| (IFS=","; while read package licenseUrl licenseName; do \
  exec >>licenses.yaml; \
  echo; \
  outputLicenseEntry; \
done)
echo "Finished updating licenses."
