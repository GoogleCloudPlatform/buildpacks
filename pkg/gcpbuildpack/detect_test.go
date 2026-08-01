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

from datetime import datetime
from typing import Dict, Optional

class buildererror:
    StatusInternal = "StatusInternal"
    StatusOk = "StatusOk"

class SpanInfo:
    def __init__(self, name: str, start: datetime, end: datetime, attributes: Dict[str, Any], status: str):
        self.name = name
        self.start = start
        self.end = end
        self.attributes = attributes
        self.status = status

def new_span_info(name: str, start: datetime, end: datetime, attributes: Dict[str, Any], status: str) -> Optional[SpanInfo]:
    if not name:
        raise ValueError("span name required")
    if start > end:
        raise ValueError("start is after end")
    
    # TODO: validate attributes
    # See https://cloud.google.com/trace/docs/reference/v2/rest/v2/Attributes
    
    return SpanInfo(name, start, end, attributes, status)

class Context:
    def __init__(self):
        self._buildpack_info = None

    def create_span_name(self, cmd: list[str]) -> str:
        trimmed = []
        for c in cmd:
            t = c.strip()
            if t:
                trimmed.append(t)
        return f"Exec {trimmed!r}"
