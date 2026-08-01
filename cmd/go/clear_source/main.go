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

"""
Tests for the clear_source buildpack library functions.
"""

import os
from unittest import mock

import pytest

from clear_source.lib import detect_fn


@pytest.mark.parametrize(
    "test_case",
    [
        {
            "name": "env var set",
            "env_vars": {"GOOGLE_CLEAR_SOURCE": "true"},
            "expected_result_code": 0,
        },
        {
            "name": "GOOGLE_CLEAR_SOURCE not set",
            "env_vars": {},
            "expected_result_code": 100,
        },
        {
            "name": "GOOGLE_CLEAR_SOURCE set and devmode enabled",
            "env_vars": {
                "GOOGLE_CLEAR_SOURCE": "true",
                "GOOGLE_DEVMODE": "true",
            },
            "expected_result_code": 100,
        },
    ],
)
def test_detect(test_case):
    """Test the detect function with various environment scenarios."""
    env_vars = test_case["env_vars"]
    expected_result_code = test_case["expected_result_code"]

    # Mock os.environ for testing
    with mock.patch.dict(os.environ, env_vars, clear=True):
        result, _ = detect_fn(None)

    if expected_result_code == 0:
        assert result is not None and result.opt_in is True
    else:
        assert result is None
