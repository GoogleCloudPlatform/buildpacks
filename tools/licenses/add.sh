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

# The add.sh scripts adds licenses for dependencies to the specified image.
#
# Usage:
#   ./add.sh <image>

set -euo pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_DIR="$(cd "${DIR}/../.." && pwd)"

IMAGE="${1:-}"

if [[ -z "${IMAGE}" ]]; then
  echo "Usage: $0 <image>"
  exit 1
fi


TEMP="$(mktemp -d)"
trap "rm -rf $TEMP" EXIT

echo "Gathering licenses into $TEMP"
echo
"${DIR}/gather.sh" "$TEMP"

LIFECYCLE_VERSION="$(docker inspect "$IMAGE" --format='{{(index (index .Config.Labels) "io.buildpacks.builder.metadata")}}' | jq -r '.lifecycle.version')"

echo "Adding licenses to $IMAGE"
cat <<EOF > "$TEMP/Dockerfile"
FROM "$IMAGE"
COPY files/buildpacks /usr/local/share/licenses/buildpacks/
COPY files/lifecycle-v$LIFECYCLE_VERSION/ /usr/local/share/licenses/lifecycle/
EOF

docker build -t "$IMAGE" "$TEMP"
