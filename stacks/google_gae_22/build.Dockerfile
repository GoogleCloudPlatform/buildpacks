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

FROM ubuntu:22.04

# Version identifier of the image.
ARG CANDIDATE_NAME

COPY build-packages.txt /tmp/packages.txt
RUN \
  # Write version information
  mkdir -p /usr/local/versions && \
    echo ${CANDIDATE_NAME} > /usr/local/versions/build_base && \
  # Disable universe and multiverse repositories
  mv /etc/apt/sources.list /etc/apt/sources.list.universe && \
  cat /etc/apt/sources.list.universe \
    | sed 's/^deb\(.*\(multi\|uni\)verse\)/# deb\1/g' >/etc/apt/sources.list && \
  # Install packages
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get update -y && \
  apt-get upgrade -y --no-install-recommends --allow-remove-essential && \
  xargs -a /tmp/packages.txt \
    apt-get -y -qq --no-install-recommends --allow-remove-essential install && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  rm /tmp/packages.txt && \
  unset DEBIAN_FRONTEND && \
  # Restore universe and multiverse repositories to ease extending our stacks
  mv /etc/apt/sources.list.universe /etc/apt/sources.list && \
  # Configure the system locale
  locale-gen en_US.UTF-8 && \
  update-locale LANG=en_US.UTF-8 LANGUAGE=en_US.UTF-8 LC_ALL=en_US.UTF-8 && \
  # Configure the user's home directory
  mkdir /www-data-home && \
  chown www-data:www-data /www-data-home && \
  usermod -d /www-data-home www-data

ARG cnb_uid=33
ARG cnb_gid=33
USER ${cnb_uid}:${cnb_gid}

CMD []
ENTRYPOINT []

ENV CNB_STACK_ID="google.gae.22"
ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_guid}

# Standard buildpacks metadata
LABEL io.buildpacks.stack.id="google.gae.22"
LABEL io.buildpacks.stack.distro.name="Ubuntu"
LABEL io.buildpacks.stack.distro.version="22.04"
LABEL io.buildpacks.stack.maintainer="Google"
LABEL io.buildpacks.stack.mixins="[]"
LABEL io.buildpacks.stack.homepage \
  "https://github.com/GoogleCloudPlatform/buildpacks/stacks/google-gae-22"