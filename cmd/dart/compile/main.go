# Copyright 2025 Google LLC
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

"""Implements dart/compile buildpack. The compile buildpack runs dart compile to produce a self-contained executable."""

from cmd.dart.compile import lib
import gcpbuildpack as gcp

if __name__ == "__main__":
    gcp.main(lib.detect_fn, lib.build_fn)
