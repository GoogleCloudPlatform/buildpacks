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

# This script reads the memory limit of the current container from the
# memory.limit_in_bytes cgroup file, then configures Nodejs to use 80% of the
# of the available memory for the old space using the NODE_OPTIONS environment
# variable. The script is designed to be used as a buildpack exec.d script so
# we set the env var by writing to the third file descriptor. For more details
# see https://buildpacks.io/docs/for-buildpack-authors/how-to/write-buildpacks/use-exec.d/

LIMIT_BYTES=""

# Check for Cgroup v1
if [ -f /sys/fs/cgroup/memory/memory.limit_in_bytes ]; then
  LIMIT_BYTES=$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes)
# Check for Cgroup v2
elif [ -f /sys/fs/cgroup/memory.max ]; then
  LIMIT_BYTES=$(cat /sys/fs/cgroup/memory.max)
fi

# If a limit was found, and it's not the v2 "max" string, calculate and set NODE_OPTIONS
if [[ -n "$LIMIT_BYTES" && "$LIMIT_BYTES" != "max" ]]; then
  TOTAL_MEM_MB=$(echo "$LIMIT_BYTES" | awk '{printf "%d", $1 / 1024 / 1024}')
  HEAP_LIMIT_MB=$(($TOTAL_MEM_MB * 80 / 100)) # use 80% of available memory
  if [[ -z "${NODE_OPTIONS}" ]]; then
    # NODE_OPTIONS is not defined so we just set it
    echo "NODE_OPTIONS=\"--max-old-space-size=$HEAP_LIMIT_MB\"" >&3
  else
    if [[ $NODE_OPTIONS != *"max-old-space-size"* ]]; then
      # NODE_OPTIONS is already defined so we append the --max-old-space-size
      echo "NODE_OPTIONS=\"${NODE_OPTIONS//\"/\\\"} --max-old-space-size=$HEAP_LIMIT_MB\"" >&3
    fi
  fi
fi