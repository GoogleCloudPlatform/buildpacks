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

# The update-licenses.sh script updates licenses for all dependencies.
#
# The scripts updates:
#   licenses/licenses.yaml  a file containing dependecy names and licenses.
#   licenses/licenses/...   a directory containing all license files.
#
# Usage:
#   update-licenses.sh

set -euo pipefail

# Version of lifecycle to get licenses for.
LIFECYCLE_VERSION="v0.7.3"

DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_DIR="$(cd "${DIR}/.." && pwd)"
LICENSES_DIR="${PROJECT_DIR}/licenses"
LICENSEFILES_DIR="${LICENSES_DIR}/files"

# Perform all operations relative to $PROJECT_DIR.
cd "${PROJECT_DIR}"

# Use github.com/google/go-licenses to extract third-party licenses for
# Go libraries used in this project to licenses/, and to build a summary
# licenses.yaml.

# Require and download lifecycle separately because go-licenses does not support
# the @version notation.
go get "github.com/buildpacks/lifecycle@${LIFECYCLE_VERSION}"

# Extract Go licenses for third-party libraries used in the GCP Buildpacks
# and other binaries included in the builder images like the CNB Lifecycle.
# Use --force to cause the save_path to be deleted.
echo "Saving license files to ${LICENSEFILES_DIR}"
go run github.com/google/go-licenses save --force \
  --save_path "${LICENSEFILES_DIR}" \
  "${PROJECT_DIR}/..." \
  github.com/buildpacks/lifecycle

# Build licenses.yaml.
cat > "${LICENSES_DIR}/licenses.yaml" << EOF
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
  if [ -f "${LICENSEFILES_DIR}/${license_path}" ]; then
    echo "  license_path: /usr/local/share/licenses/${license_path}"
  elif [ -f "${LICENSEFILES_DIR}/${package}/LICENSE" ]; then
    echo "  license_path: /usr/local/share/licenses/${package}/LICENSE"
  elif [ -f "${LICENSEFILES_DIR}/${package}/COPYING" ]; then
    echo "  license_path: /usr/local/share/licenses/${package}/COPYING"
  elif [ -f "${LICENSEFILES_DIR}/${package}/NOTICE" ]; then
    echo "  license_path: /usr/local/share/licenses/${package}/NOTICE"
  else
    echo "ERROR: unable to find license file for ${package}" 1>&2
    exit 1
  fi
}

# example output from `wgo-licenses csv`:
#   github.com/blang/semver,https://github.com/blang/semver/blob/master/LICENSE,MIT
echo "Writing ${LICENSES_DIR}/licenses.yaml"
echo "Note: it is expected to see 'error discovering URL' messages for go modules"
go run github.com/google/go-licenses csv \
  "${PROJECT_DIR}/..." \
  github.com/buildpacks/lifecycle \
| sort \
| (IFS=","; while read package licenseUrl licenseName; do \
  exec >>"${LICENSES_DIR}/licenses.yaml"; \
  echo; \
  outputLicenseEntry; \
done)
echo "Finished updating licenses."
