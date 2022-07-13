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

# The buildpack-package.sh script creates a meta buildpack in the cnb file
# format.
#
# NOTE: this script sets a HOME folder. This is necessary because pack requires
# a ${HOME} to be defined.
#
# Note: This script should not be invoked directly; it is used in defs.bzl.
#
# Usage:
# ./buildpack-package.sh []args

set -euox pipefail

readonly output_file="${1:?output path missing}"; shift
readonly package_toml="${1:?package toml path missing}"; shift

# Blaze does not set $HOME which is required by pack.
export HOME="$(pwd)"

pack buildpack package "${output_file}" --format file --config "${package_toml}"
