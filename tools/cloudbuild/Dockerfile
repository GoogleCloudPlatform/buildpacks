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


# Cloud Build builder that extends the bazel builder with pack and
# container-structure-test binaries, as well as the Go toolchain.

FROM gcr.io/cloud-builders/bazel

# Binaries must be in /usr/bin to be visible to Bazel.
ARG PACK_VERSION=0.10.0
RUN curl -fsSL https://storage.googleapis.com/container-structure-test/latest/container-structure-test-linux-amd64 -o /usr/bin/container-structure-test && chmod +x /usr/bin/container-structure-test
RUN curl -fsSL https://github.com/buildpacks/pack/releases/download/v${PACK_VERSION}/pack-v${PACK_VERSION}-linux.tgz | tar xz -C /usr/bin
RUN curl -fsSL https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 -o /usr/bin/jq && chmod +x /usr/bin/jq

# Go is required for go-licenses even when statically compiled.
RUN mkdir -p /go \
  && mkdir -p /usr/local/go \
  && curl --silent https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz \
  | tar xz --strip-components=1 --directory /usr/local/go
ENV GOPATH=/go
ENV GOROOT=/usr/local/go/
ENV PATH=/go/bin:/usr/local/go/bin:${PATH}
RUN go get github.com/google/go-licenses
