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

import hashlib
import json
from dataclasses import dataclass
from typing import Any

error_id_length = 8


@dataclass
class Error:
    BuildpackID: str
    BuildpackVersion: str
    Type: "Status"
    Status: "Status"
    ID: str
    Message: str
    internal_error: Exception

    def __str__(self) -> str:
        return f"(error ID: {self.ID}):\n{self.Message}"

    def unwrap(self) -> Exception:
        return self.internal_error


def errorf(status: "Status", format_str: str, *args: Any) -> Error:
    message = format_str % args
    error_id = generate_error_id(message)
    internal_error = Exception(message)
    return Error(
        BuildpackID="",
        BuildpackVersion="",
        Type=status,
        Status=status,
        ID=error_id,
        Message=message,
        internal_error=internal_error,
    )


def internal_errorf(format_str: str, *args: Any) -> Error:
    return errorf(Status.INTERNAL, format_str, *args)


def user_errorf(format_str: str, *args: Any) -> Error:
    return errorf(Status.UNKNOWN, format_str, *args)


def generate_error_id(*parts: str) -> str:
    h = hashlib.sha256()
    for part in parts:
        h.update(part.encode())
    result = h.hexdigest()[:error_id_length]
    return result.lower()
