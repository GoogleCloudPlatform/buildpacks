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

# nginx creates the following directories at bootup, even if configured to be
# ignored. Because of the read-only filesystem (for cnb user), we create them
# ahead of time.
RUN bash -c 'mkdir -p /var/lib/nginx/{body,proxy,fastcgi,uwsgi,scgi}'
RUN bash -c 'mkdir -p /tmp/{var,run,doc_root}'
RUN bash -c 'mkdir -p /var/log/nginx'
RUN bash -c 'mkdir -p /tmp/var/cache/nginx'
RUN chmod -R 777 /var/lib/nginx/
RUN chmod -R 777 /var/log/nginx/
RUN chmod -R 777 /tmp/

ENV PORT 8080
USER cnb
