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

COPY run-packages.txt /tmp/packages.txt
RUN \
  # Write version information
  mkdir -p /usr/local/versions && \
    echo ${CANDIDATE_NAME} > /usr/local/versions/run_base && \
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
  # Install the start and serve binaries.
  mkdir /usr/lib/pid1 && mkdir /usr/lib/serve && \
  curl -A GCPBuildpacks https://dl.google.com/runtimes/serve/serve-1.0.0.tar.gz \
    | tar xvz -C /usr/lib/serve && \
  curl -A GCPBuildpacks https://dl.google.com/runtimes/pid1/pid1-1.0.0.tar.gz \
    | tar xvz -C /usr/lib/pid1 && \
  # These invoked as start, serve, /start and /serve in some places.
  cp /usr/lib/pid1/pid1 /usr/bin/pid1 && \
  cp /usr/lib/serve/main /usr/bin/serve && \
  ln -s /usr/bin/pid1 /start && \ 
  ln -s /usr/bin/pid1 /usr/bin/start && \
  ln -s /usr/bin/serve /serve && \
  # Configure the user's home directory
  mkdir /www-data-home && \
  chown www-data:www-data /www-data-home && \
  usermod -d /www-data-home www-data && \
  # Users added to this group may have additional permissions on non-critical
  # gofer filesystems, such as destinations for internal logs and metrics.
  groupadd --system google && \
  # Pack hardcodes /workspace; we symlink it to /srv because that is the working
  # directory set in the GAE instance definition.
  rm -rf /srv && ln -s /workspace /srv && \
  # /home and /root must be writeable.
  rm -rf /home && ln -sT /tmp /home && \
  rm -rf /root && ln -sT /tmp /root && \
  # Nginx creates the following directories at bootup, even if configured to be
  # ignored. Because of the read-only filesystem in production, we create them
  # ahead of time.
  bash -c 'mkdir -p /var/lib/nginx/{body,proxy,fastcgi,uwsgi,scgi}' && \
  mkdir -m 0777 /tmp/run && \
  rm -rf /run && ln -s /tmp/run /run && \
  rm -rf /var/cache/nginx && ln -s /tmp/var/cache/nginx /var/cache/nginx && \
  # Remove /var/log/* because nginx attempts to write to it.
  chmod a+w /var/log && \
  rm -rf /var/log/*

# Workdir must be set to the resolved symlink path. While Docker resolves the
# symlink at startup, the App Engine dataplane reads the workdir from the image
# manifest and sets it as PWD in the environment of the created container. This
# This causes issues in some languages, e.g. Go, which returns the value of PWD
# from os.Getwd() if set. If PWD is a symlink, filepath.Walk() with os.Getwd()
# will not return any files since it does not follow symlinks.
WORKDIR /workspace

ARG cnb_uid=33
ARG cnb_gid=33
USER ${cnb_uid}:${cnb_gid}

ENV CNB_STACK_ID="google.gae.22"
ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}

# Standard buildpacks metadata.
LABEL io.buildpacks.stack.id="google.gae.22"
LABEL io.buildpacks.stack.distro.name="Ubuntu"
LABEL io.buildpacks.stack.distro.version="22.04"
LABEL io.buildpacks.stack.maintainer="Google"
LABEL io.buildpacks.stack.mixins="[]"
LABEL io.buildpacks.stack.homepage \
  "https://github.com/GoogleCloudPlatform/buildpacks/stacks/google-gae-22"

# Set $PORT to 8080 by default
ENV PORT 8080
EXPOSE 8080

CMD []
ENTRYPOINT []