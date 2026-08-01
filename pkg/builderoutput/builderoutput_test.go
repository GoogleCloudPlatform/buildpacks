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

import json
from dataclasses import dataclass, field
from typing import List, Optional

from pkg.buildererror import Error as BuilderError, Status
from pkg.buildermetadata import BuilderMetadata
from pkg.buildermetrics import BuilderMetrics


@dataclass
class BuilderStat:
    buildpack_id: str = ""
    buildpack_version: str = ""
    duration_ms: int = 0
    user_duration_ms: int = 0

    @classmethod
    def from_json(cls, data: dict):
        return cls(
            buildpack_id=data.get("buildpackId", ""),
            buildpack_version=data.get("buildpackVersion", ""),
            duration_ms=data.get("totalDurationMs", 0),
            user_duration_ms=data.get("userDurationMs", 0)
        )


@dataclass
class BuilderOutput:
    installed_runtime_versions: List[str] = field(default_factory=list)
    metrics: BuilderMetrics = field(default_factory=BuilderMetrics)
    error: Optional[BuilderError] = None
    metadata: BuilderMetadata = field(default_factory=BuilderMetadata)
    stats: List[BuilderStat] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)
    custom_image: bool = False

    @classmethod
    def from_json(cls, json_bytes: bytes) -> 'BuilderOutput':
        data = json.loads(json_bytes.decode('utf-8'))
        
        return cls(
            installed_runtime_versions=data.get("rtVersions", []),
            metrics=BuilderMetrics.from_json(data.get("metrics", {})),
            error=BuilderError.from_json(data.get("error", {})) if "error" in data else None,
            metadata=BuilderMetadata.from_json(data.get("metadata", {})),
            stats=[BuilderStat.from_json(stat) for stat in data.get("stats", [])],
            warnings=data.get("warnings", []),
            custom_image=data.get("customImage", False)
        )

    def to_json(self) -> bytes:
        return json.dumps({
            "rtVersions": self.installed_runtime_versions,
            "metrics": self.metrics.to_dict(),
            "error": self.error.to_dict() if self.error else {},
            "metadata": self.metadata.to_dict(),
            "stats": [stat.__dict__ for stat in self.stats],
            "warnings": self.warnings,
            "customImage": self.custom_image
        }).encode('utf-8')

    @staticmethod
    def new() -> 'BuilderOutput':
        return BuilderOutput(
            metrics=BuilderMetrics.new(),
            metadata=BuilderMetadata.new()
        )

    def is_system_error(self) -> bool:
        return self.error and self.error.type == Status.INTERNAL
