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

source "https://rubygems.org"
gem "sinatra", "~> 2.2.0"
gem "toys-core", git: "https://github.com/dazuma/toys", ref: "16a06e5e1bb394c89dbc23c70f628532e7a8fcb9"
gem "webrick", "~> 1.7"
gem "base64", "~> 0.2"
