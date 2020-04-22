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

# The presubmit.sh script runs acceptance tests for tests matching $FILTER.
#
# The value of FILTER is passed to Blaze query and used to select a subset of
# targets. Tests are split into smaller sets that can run concurrently on
# different machines.

set -euo pipefail

if [[ -v KOKORO_ARTIFACTS_DIR ]]; then
  cd ${KOKORO_ARTIFACTS_DIR}/git/buildpacks
  use_bazel.sh latest

  # Move docker's storage location to scratch disk so we don't run out of space.
  echo 'DOCKER_OPTS="${DOCKER_OPTS} --data-root=/tmpfs/docker"' | sudo tee --append /etc/default/docker
  sudo service docker restart
fi

if [[ ! -v FILTER ]]; then
  echo 'Must specify $FILTER.'
  exit 1
fi

readonly PACK_VERSION="0.10.0"

temp="$(mktemp -d)"
curl -fsSL "https://storage.googleapis.com/container-structure-test/latest/container-structure-test-linux-amd64" -o "$temp/container-structure-test" && chmod +x "$temp/container-structure-test"
curl -fsSL "https://github.com/buildpacks/pack/releases/download/v${PACK_VERSION}/pack-v${PACK_VERSION}-linux.tgz" | tar xz -C "$temp"

# TODO: Build stack images when testing GCP builder instead of pulling.
# TODO: Process logs to be parseable by Sponge.

# Filter out non-test targets, such as builder.image rules.
bazel test --test_output=errors --action_env="PATH=$temp:$PATH" $(bazel query "filter('${FILTER}', kind('.*_test rule', '//builders/...'))")
