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

# Gather license files in a tar as Docker's ADD preserves file ownership and mod times.
# The tar must be located within the docker build context (${DIR})
echo "> Gathering licenses"
LICENSES_BASE="licenses"
LICENSES_TAR="licenses.tar"
trap "rm -rf \"${DIR}/${LICENSES_TAR}\"" EXIT
tar -cf "${DIR}/${LICENSES_TAR}" --mode="a=rX" --mtime="@0" --owner=0 --group=0 --sort=name -C "${DIR}/${LICENSES_BASE}" .
tar -rf "${DIR}/${LICENSES_TAR}" --mode="a=rX" --mtime="@0" --owner=0 --group=0 -C "${DIR}" licenses.yaml

echo "> Building gcp/base common image"
docker build -t "common" -f "${DIR}/Dockerfile.parent" "${DIR}"
echo "> Building gcr.io/buildpacks/gcp/run"
docker build --build-arg "from_image=common" -t "gcr.io/buildpacks/gcp/run" -f "${DIR}/Dockerfile.run" "${DIR}"
echo "> Building gcr.io/buildpacks/gcp/build"
docker build --build-arg "from_image=common" -t "gcr.io/buildpacks/gcp/build" -f "${DIR}/Dockerfile.build" "${DIR}"

echo "> Validating gcr.io/buildpacks/gcp/build"
container-structure-test test -c build-test.yaml -i gcr.io/buildpacks/gcp/build:latest
