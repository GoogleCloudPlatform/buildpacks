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

import sys
from pip._internal.operations import freeze


def testFunction(request):
  # Checks full list of pip packages installed via pip freeze.
  length = len(list(freeze.freeze()))
  # Python 3.7 contains extra compat dependencies. If this test begins to fail,
  # it means another transitive dependency has been added. This transitive
  # dependency should be explicitly pinned in the functions_framework and
  # functions_framework_compat buildpacks and the number here should be updated.
  if sys.version_info.minor == 7:
    return 'PASS' if length == 54 else 'FAIL: received ' + str(length)
  else:
    return 'PASS' if length == 16 else 'FAIL: received ' + str(length)
