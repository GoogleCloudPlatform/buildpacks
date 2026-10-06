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

FROM gcr.io/gcp-runtimes/ubuntu_18_0_4

ARG cnb_uid=1000
ARG cnb_gid=1000
ARG stack_id="google"

# Required by python/runtime: libexpat1, libffi6, libmpdecc2.
# Required by dotnet/runtime: libicu60
# Required by go/runtime: tzdata (Go may panic without /usr/share/zoneinfo)
# Required by ruby/runtime: libyaml
# Required by php/runtime: libtidy5, libpq5, libxml2, libenchant1c2a,
# libpng16-16, libonig4, libjpeg8, libfreetype6, libxslt1.1, libzip4, libsodium23,
RUN apt-get update -y && \
  apt-get upgrade -y --no-install-recommends --allow-remove-essential && \
  apt-get install -y --no-install-recommends --allow-remove-essential \
    libexpat1 \
    libffi6 \
    libmpdec2 \
    libicu60 \
    libc++1-9 \
    tzdata \
    libyaml-0-2 \
    libtidy5 \
    libpq5 \
    libxml2 \
    libenchant1c2a \
    libpng16-16 \
    libonig4 \
    libjpeg8 \
    libfreetype6 \
    libsodium23 \
    libxslt1.1 \
    libzip4 \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

LABEL io.buildpacks.stack.id=${stack_id}

RUN groupadd cnb --gid ${cnb_gid} && \
  useradd --uid ${cnb_uid} --gid ${cnb_gid} -m -s /bin/bash cnb

ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}
ENV CNB_STACK_ID=${stack_id}
