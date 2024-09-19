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
# Note: This script is meant to be invoked using Blaze/Bazel.
#
# Usage:
#   ./pull-images <product> <runtime>
#
# Pulls or builds stack images required by builders/<product>/<runtime>.

readonly product="${1:?product name missing}"
readonly runtime="${2:?runtime name missing}"
readonly project="gae-runtimes"
readonly candidate="latest"

echo "Pulling stack images for ${product}/${runtime}"
if [[ "${product}" == "gcp" ]]; then
  # We pull both the google-22 and the google-18 images, because the OSS builder
  # supports both.
  docker pull "gcr.io/buildpacks/${product}/run:${candidate}"
  docker pull "gcr.io/buildpacks/${product}/build:${candidate}"
  docker pull "gcr.io/buildpacks/google-22/build:${candidate}"
  docker pull "gcr.io/buildpacks/google-22/run:{candidate}"
else
  docker pull "gcr.io/${project}/buildpacks/${runtime}/run:${candidate}"
  docker pull "gcr.io/${project}/buildpacks/${runtime}/build:${candidate}"
fi
