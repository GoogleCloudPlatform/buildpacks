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

"""Simple flask web server used in acceptance tests.
"""
import os
import sys

from flask import Flask
from flask import request

app = Flask(__name__)


@app.route("/")
def hello():
  return "PASS"


@app.route("/version")
def version():
  """Verify that the script is run using the correct version of the interpreter.

  Returns:
    String representing the response body.
  """
  want = request.args.get("want")
  if not want:
    return "FAIL: ?want must be set to a version"

  got = sys.version
  if not got.startswith(want):
    return 'FAIL: "{}" does not start with "{}"'.format(got, want)

  return "PASS"

if __name__ == "__main__":
  app.run(port=os.environ["PORT"], debug=True)
