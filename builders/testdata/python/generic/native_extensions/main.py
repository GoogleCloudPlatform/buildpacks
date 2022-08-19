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

from flask import Flask
import numpy as np
from numpy.ctypeslib import load_library

app = Flask(__name__)


def test_multiarray_umath():
  try:
    # Should succeed
    load_library("_multiarray_umath", np.core._multiarray_umath.__file__)
  except ImportError as e:
    msg = ("Import error was: %s" % str(e))
    print(msg)
    return e


@app.route("/")
def hello():
  if test_multiarray_umath() is None:
    return "PASS"
  else:
    return "FAILED"

if __name__ == "__main__":
  app.run(port=os.environ["PORT"], debug=True)

