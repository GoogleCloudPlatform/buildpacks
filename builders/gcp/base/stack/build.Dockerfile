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
ADD licenses.tar /usr/local/share/licenses

# Required to install nodejs/runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
  xz-utils \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Required for some project dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
  git \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Required by many use-cases.
RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Required to extract gradle.
RUN apt-get update && apt-get install -y --no-install-recommends \
  unzip \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

USER cnb
