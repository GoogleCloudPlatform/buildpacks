# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest
from unittest.mock import patch

import google.buildpacks.internal.buildpacktest as buildpacktest
from ..lib import detect_fn

class TestDetect(unittest.TestCase):
    def test_detect(self):
        test_cases = [
            {
                "name": "csproj",
                "files": {"Program.cs": "", "app.csproj": ""},
                "env": {},
                "want": 0,
            },
            {
                "name": "fsproj",
                "files": {"Program.fs": "", "app.fsproj": ""},
                "env": {},
                "want": 0,
            },
            {
                "name": "vbproj",
                "files": {"Program.vb": "", "app.vbproj": ""},
                "env": {},
                "want": 0,
            },
            {
                "name": "with_build_env",
                "files": {"Program.cs": ""},
                "env": {"GOOGLE_BUILDABLE": "myapp"},
                "want": 0,
            },
            {
                "name": "project_and_build_env",
                "files": {"Program.cs": "", "app.csproj": ""},
                "env": {},
                "want": 0,
            },
            {
                "name": "unsupported_pyproj",
                "files": {".pyproj": ""},
                "env": {},
                "want": 100,
            },
            {
                "name": "unsupported_partly_matching",
                "files": {"Program.cs": "", "app.mycsproj": ""},
                "env": {},
                "want": 100,
            },
            {
                "name": "no_project_or_env",
                "files": {"Program.cs": ""},
                "env": {},
                "want": 100,
            },
        ]

        for case in test_cases:
            with self.subTest(name=case["name"]):
                with buildpacktest.TestDetect() as t:
                    t.set_files(case["files"])
                    if case.get("env"):
                        os.environ.update(case["env"])
                    result, err = detect_fn(t.ctx)
                    if err:
                        self.fail(f"Unexpected error: {err}")
                    self.assertEqual(result.status_code, case["want"])

if __name__ == "__main__":
    unittest.main()
