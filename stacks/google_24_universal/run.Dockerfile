# Copyright 2024 Google LLC
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

FROM marketplace.gcr.io/google/ubuntu2404:latest

ARG cnb_uid=33
ARG cnb_gid=33

COPY run-packages.txt /tmp/packages.txt

# Version identifier of the image.
ARG CANDIDATE_NAME
RUN --mount=type=secret,id=pro-attach-config,target=/etc/secrets/pro-attach-config \
  apt-get update && \
  # Here we install `pro` (ubuntu-advantage-tools) as well as ca-certificates,
  # which is required to talk to the Ubuntu Pro authentication server securely.
  apt-get install --no-install-recommends -y ubuntu-advantage-tools \
  ca-certificates && \
  (pro attach --attach-config /etc/secrets/pro-attach-config || true) && \
  # Write version information
  mkdir -p /usr/local/versions && \
    echo ${CANDIDATE_NAME} > /usr/local/versions/run_base && \
  # Disable multiverse repositories
  mv /etc/apt/sources.list /etc/apt/sources.list.universe && \
  cat /etc/apt/sources.list.universe \
    | sed 's/^deb\(.*\multiverse\)/# deb\1/g' >/etc/apt/sources.list && \
  # Install packages
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get update -y && \
  apt-get upgrade -y --no-install-recommends --allow-remove-essential && \
  xargs -a /tmp/packages.txt \
    apt-get -y -qq --no-install-recommends --allow-remove-essential install && \
  apt-get purge --auto-remove -y ubuntu-advantage-tools && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  rm /tmp/packages.txt && \
  unset DEBIAN_FRONTEND && \
  # Restore multiverse repositories to ease extending our stacks
  mv /etc/apt/sources.list.universe /etc/apt/sources.list && \
  # Configure the system locale
  locale-gen en_US.UTF-8 && \
  update-locale LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8 && \
  # Configure the user
  groupadd cnb --gid ${cnb_gid} && \
  useradd --uid ${cnb_uid} --gid ${cnb_gid} -m -s /bin/bash cnb && \
  # Nothing in this image requires an additional license, but we need to add an
  # empty license.yaml so the license validation doesn't fail.
  mkdir -p /.google/usr/local/share/licenses/base_runtime/ && \
  touch /.google/usr/local/share/licenses/base_runtime/licenses.yaml && \
  # Remove /var/log/* because nginx attempts to write to it.
  chmod a+w /var/log && \
  rm -rf /var/log/*

# Install the following packages separately as putting them in packages.txt 
# will cause "unable to locate packages" error
RUN export DEBIAN_FRONTEND=noninteractive && \
  apt-get update -y && \
  apt-get upgrade -y --no-install-recommends --allow-remove-essential && \
  apt-get -y -qq --no-install-recommends --allow-remove-essential install \
    libzip4

USER cnb

ENV LANG="en_US.UTF-8"
ENV LANGUAGE="en_US:en"
ENV LC_ALL="en_US.UTF-8"
ENV CNB_STACK_ID="google.24"
ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}

# Standard buildpacks metadata
LABEL io.buildpacks.stack.id="google.24"
LABEL io.buildpacks.stack.distro.name="Ubuntu"
LABEL io.buildpacks.stack.distro.version="24.04"
LABEL io.buildpacks.stack.maintainer="Google"
LABEL io.buildpacks.stack.mixins="[]"
LABEL io.buildpacks.stack.homepage \ 
  "https://github.com/GoogleCloudPlatform/buildpacks/stacks/google_24_universal"

# Set $PORT to 8080 by default
ENV PORT 8080
EXPOSE 8080