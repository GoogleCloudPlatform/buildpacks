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

"""Tests for nodejs/appengine buildpack."""

import os
from unittest import mock

import pytest

from buildpacks.gcpbuildpack.testhelpers import (
    create_context,
    setup_files_and_env,
)
from . import lib


@pytest.fixture
def context():
    """Fixture to create a test context."""
    return create_context()


@pytest.fixture
def state():
    """Fixture to create a test state."""
    return {"files": {}, "env": {}}


def test_detect(context, state):
    """Test the detect function with various scenarios."""
    test_cases = [
        {
            "name": "with package",
            "files": {"index.js": "", "package.json": ""},
            "env": {"X_GOOGLE_TARGET_PLATFORM": "gae"},
            "want": 0,
        },
        {
            "name": "without package",
            "files": {"index.js": ""},
            "env": {"X_GOOGLE_TARGET_PLATFORM": "gae"},
            "want": 0,
        },
        {
            "name": "without package, without GAE target platform",
            "files": {"index.js": ""},
            "env": {},
            "want": 100,
        },
    ]

    for case in test_cases:
        with pytest.subTest(case["name"]):
            setup_files_and_env(context, state, case["files"], case["env"])
            
            with mock.patch.dict(os.environ, case.get("env", {})):
                result_code, _ = lib.detect_fn(context)
                assert result_code == case["want"]
