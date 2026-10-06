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

# Java launcher that configures memory based on how much is available.
# If the environment variable $JAVA_TOOL_OPTIONS is set (even if empty), we just
# launch the java command we are given. If it is not set, and we have a hint
# from the environment of what the memory size is, then we set JAVA_TOOL_OPTIONS
# to a value that should cause the JVM to use 70% of that memory size.
# The JVM reads command-line options from this environment variable.
if [[ -z "${JAVA_TOOL_OPTIONS+x}" && -n "${X_GOOGLE_MEMORY_HINT_DO_NOT_USE}" ]]
then
  export JAVA_TOOL_OPTIONS="-XX:MaxRAM=${X_GOOGLE_MEMORY_HINT_DO_NOT_USE}m -XX:MaxRAMPercentage=70"
fi
exec "$@"
