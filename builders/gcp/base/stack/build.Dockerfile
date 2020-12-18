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

ARG from_image
FROM ${from_image}
COPY licenses/ /usr/local/share/licenses/buildpacks/

# build-essential is required by many usecases.
# git is required for some project dependencies.
# python3 is required by node-gyp to compile native modules.
# unzip is required to extract gradle.
# xz-utils is required to install nodejs/runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  git \
  python3 \
  unzip \
  xz-utils \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

USER cnb
