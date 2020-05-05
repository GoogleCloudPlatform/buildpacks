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

# The build.sh script builds stack images for the gcp/base builder.
#
# The script builds the following two images:
#   gcr.io/buildpacks/gcp/run
#   gcr.io/buildpacks/gcp/build
#
# It also validates that the build image includes all required licenses.
#
# Usage:
#   ./build.sh

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "> Building gcp/base common image"
docker build -t "common" -f "${DIR}/parent.Dockerfile" "${DIR}"
echo "> Building gcr.io/buildpacks/gcp/run"
docker build --build-arg "from_image=common" -t "gcr.io/buildpacks/gcp/run" -f "${DIR}/run.Dockerfile" "${DIR}"
echo "> Building gcr.io/buildpacks/gcp/build"
docker build --build-arg "from_image=common" -t "gcr.io/buildpacks/gcp/build" -f "${DIR}/build.Dockerfile" "${DIR}"

echo "> Validating gcr.io/buildpacks/gcp/build"
container-structure-test test -c "${DIR}/build-test.yaml" -i gcr.io/buildpacks/gcp/build:latest
