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

# The diff.sh script diffs submitted licenses with the expected state.
#
# Usage:
#   ./diff.sh
#
# Exits with 0 if there is no diff, otherwise 1.

set -euo pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_DIR="$(cd "${DIR}/../.." && pwd)"

OLD_LICENSES="${PROJECT_DIR}/licenses"
NEW_LICENSES="$(mktemp -d)"

echo "Gathering new licenses into $NEW_LICENSES"
echo
"${DIR}/gather.sh" "$NEW_LICENSES"
echo

# Licenses files are not submitted, only keep yaml files.
echo "Removing license files"
rm -rf "$NEW_LICENSES/files"

echo "Diffing $NEW_LICENSES and $OLD_LICENSES"
echo
diff -r --exclude "BUILD.bazel" "$NEW_LICENSES" "$OLD_LICENSES"
# The following message will only be printed if there are no diffs.
echo "No diff found"
