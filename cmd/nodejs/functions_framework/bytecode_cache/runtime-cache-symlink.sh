#!/bin/bash
# Copyright 2025 Google LLC
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

# This script runs at application launch via exec.d.
# It checks if the bytecode cache directory name expected by Node.js in the
# current runtime environment differs from the directory name created by the
# buildpack during the build phase. This can happen if V8 flags differ
# between build and runtime, even if the flags do not invalidate cache files.
# If the names differ, it creates a symlink from the runtime-expected name
# to the build-time directory name, allowing Node.js to find and use the
# build-time cache and improve cold start performance.
# This script is best-effort and will not fail application startup if it
# encounters an error.

# If bytecode caching is not enabled, there is nothing to do.
if [[ -z "${NODE_COMPILE_CACHE}" ]]; then
  exit 0
fi

# Find non-empty build-time hash directory/directories, excluding 'exec.d'.
BUILD_HASH_DIRS=$(find "${NODE_COMPILE_CACHE}" -mindepth 1 -maxdepth 1 -type d -not -name "exec.d" -not -empty)

if [[ -z "$BUILD_HASH_DIRS" ]]; then
  echo "WARNING: No non-empty build-time cache directory found in ${NODE_COMPILE_CACHE}, skipping symlink."
  exit 0
fi

BUILD_HASH_DIRS_COUNT=$(echo "$BUILD_HASH_DIRS" | wc -l)

# sometimes there are multiple cache dirs, wherein only one is non empty.
if [[ $BUILD_HASH_DIRS_COUNT -gt 1 ]]; then
  echo "WARNING: Found $BUILD_HASH_DIRS_COUNT non-empty build-time cache directories in ${NODE_COMPILE_CACHE}, selecting largest."
  BUILD_HASH_TO_USE=$(echo "$BUILD_HASH_DIRS" | xargs du -s | sort -rh | head -n 1 | cut -f2)
  BUILD_HASH_NAME=$(basename "$BUILD_HASH_TO_USE")
else
  BUILD_HASH_NAME=$(basename "$BUILD_HASH_DIRS")
fi

# Generate a runtime cache hash name by compiling a dummy file.
# This tells us the directory name Node.js expects in this environment.
DUMMY_JS="/tmp/dummy-$$.js"
RUNTIME_CACHE_DIR="/tmp/runtime-cache-$$"
mkdir -p "$RUNTIME_CACHE_DIR"
echo "console.log('dummy');" > "$DUMMY_JS"

env -u NODE_COMPILE_CACHE node -e "require('node:module').enableCompileCache('$RUNTIME_CACHE_DIR'); require('$DUMMY_JS'); require('node:module').flushCompileCache();" >/dev/null 2>&1 || {
  echo "WARNING: Could not generate runtime cache directory, skipping symlink."
  rm -rf "$DUMMY_JS" "$RUNTIME_CACHE_DIR"
  exit 0
}

RUNTIME_HASH_DIRS_COUNT=$(find "${RUNTIME_CACHE_DIR}" -mindepth 1 -maxdepth 1 -type d | wc -l)
RUNTIME_HASH_DIRS=$(find "${RUNTIME_CACHE_DIR}" -mindepth 1 -maxdepth 1 -type d)
rm -rf "$DUMMY_JS" "$RUNTIME_CACHE_DIR"

if [[ $RUNTIME_HASH_DIRS_COUNT -ne 1 ]]; then
  echo "WARNING: Could not find unique runtime cache directory in temporary directory (found ${RUNTIME_HASH_DIRS_COUNT}), skipping symlink."
  exit 0
fi
RUNTIME_HASH_NAME=$(basename "$RUNTIME_HASH_DIRS")

# If the runtime hash differs from build-time, create a symlink from
# runtime-name -> build-time-name so Node.js finds the build-time cache.
if [[ "$BUILD_HASH_NAME" != "$RUNTIME_HASH_NAME" ]]; then
  echo "Runtime cache hash '${RUNTIME_HASH_NAME}' differs from build-time hash '${BUILD_HASH_NAME}', creating symlink."
  ln -sf "${BUILD_HASH_NAME}" "${NODE_COMPILE_CACHE}/${RUNTIME_HASH_NAME}"
else
  echo "Runtime cache hash '${RUNTIME_HASH_NAME}' matches build-time hash, no symlink needed."
fi
exit 0
