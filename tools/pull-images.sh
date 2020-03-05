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

# The pull-images.sh script pulls and retags stack images.
#
# Usage:
#   ./pull-images <product> <runtime>
#
# Pulls or builds stack images required by builders/<product>/<runtime>.

readonly product="${1:?product name missing}"
readonly runtime="${2:?runtime name missing}"
readonly project="gae-runtimes-private"
readonly candidate="nightly"

if [[ "${product}" == "gcp" ]]; then
  echo "Building stack images for ${product}/${runtime}"
  $(dirname "$0")/../builders/gcp/base/stack/build.sh
else
  echo "Pulling stack images for ${product}/${runtime}"
  docker pull "gcr.io/${project}/buildpacks/${runtime}/run:${candidate}"
  docker pull "gcr.io/${project}/buildpacks/${runtime}/build:${candidate}"
  docker tag "gcr.io/${project}/buildpacks/${runtime}/run:${candidate}" "gcr.io/gae-runtimes/buildpacks/${runtime}/run"
  docker tag "gcr.io/${project}/buildpacks/${runtime}/build:${candidate}" "gcr.io/gae-runtimes/buildpacks/${runtime}/build"
fi
