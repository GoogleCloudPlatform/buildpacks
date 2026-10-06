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

import subprocess
import sys


def testFunction(request):
  # Checks full list of pip packages installed via pip freeze.
  result = subprocess.check_output([sys.executable, '-m', 'pip', 'freeze'])

  # Check command output, remove empty lines.
  reqs = [req for req in result.decode('ascii').split('\n') if req]
  length = len(reqs)
  # Python 3.7 contains extra compat dependencies. If this test begins to fail,
  # it means another transitive dependency has been added. This transitive
  # dependency should be explicitly pinned in the functions_framework and
  # functions_framework_compat buildpacks and the number here should be updated.
  want = 24
  if sys.version_info.minor == 7:
    want = 53
  elif sys.version_info.minor >= 12 and sys.version_info.minor < 14:
    # 3.12 added setuptools==71.1.0, wheel==0.43.0
    want = 26
  elif sys.version_info.minor >= 14:
    # UV is the default package manager for python 3.14+ and it doesn't
    # add setuptools and wheel.
    want = 24
  return (
      'PASS'
      if length == want
      else f'FAIL: received {length} reqs: ' + ', '.join(reqs)
  )
