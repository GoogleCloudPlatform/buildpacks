# Copyright 2022 Google LLC
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

FROM gcr.io/buildpacks/google-22/build

ARG cnb_uid=1000
ARG cnb_gid=1000

USER root

# Version identifier of the image.
ARG CANDIDATE_NAME
RUN \
  # Write version information
  mkdir -p /usr/local/versions && \
  echo ${CANDIDATE_NAME} > /usr/local/versions/run_base 

USER cnb

ENV CNB_STACK_ID="firebase.apphosting.22"
ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}

# Standard buildpacks metadata
LABEL io.buildpacks.stack.id="firebase.apphosting.22"
LABEL io.buildpacks.stack.homepage \
  "https://github.com/GoogleCloudPlatform/buildpacks/stacks/firebase-app-hosting-22"

# Set $PORT to 8080 by default
ENV PORT 8080
EXPOSE 8080