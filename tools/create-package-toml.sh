#!/bin/bash
# Copyright 2022 Google LLC
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

# The create-package-toml.sh script creates a package.toml file with relative
# paths to all dependency buildpacks.
#
# Note: This script should not be invoked directly; it is used in defs.bzl.
#
# Usage:
# ./create-package-toml.sh []args

set -euox pipefail

readonly output_file="${1:?output path missing}"; shift
readonly buildpack_archive="${1:?buildpack archive missing}"; shift
readonly srcs=("${@:?srcs missing}")

output_file_parent_dir="$(dirname "${output_file}")"
newline=$'\n'
result="[buildpack]${newline}"
buildpack_archive_rel_path="$(realpath --relative-to "${output_file_parent_dir}" "${buildpack_archive}")"
result="${result}  uri=\"${buildpack_archive_rel_path}\"${newline}"

for dep in "${srcs[@]}"; do
  if [[ "${dep}" != ${buildpack_archive} ]]; then
    dep_rel_path="$(realpath --relative-to "${output_file_parent_dir}" "${dep}")"
    result="${result}${newline}[[dependencies]]${newline}  uri=\"${dep_rel_path}\"${newline}"
  fi
done

echo "${result}" > ${output_file}
