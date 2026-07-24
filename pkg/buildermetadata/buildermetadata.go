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

import json
from threading import Lock

class MetadataID(str):
    """The unique identifier for each metadata value."""
    pass

class MetadataValue(str):
    """The metadata value corresponding to the MetadataID."""
    pass

MetadataIDNames = {
    "1": "IsUsingGenkit",
    "2": "IsUsingGenAI", 
    "3": "FrameworkName",
    "4": "FrameworkVersion",
    "5": "AdapterName",
    "6": "AdapterVersion",
    "7": "MonorepoName",
    "8": "PackageManager",
    "9": "ConfigFile"
}

class BuilderMetadata:
    """Contains the builder metadata to be reported to RCS via BuilderOutput."""
    
    def __init__(self):
        self.metadata = {}
        
    def get_value(self, id: MetadataID) -> MetadataValue:
        """Returns the Metadata value with given ID, initializes if not present."""
        if id not in self.metadata:
            self.metadata[id] = "false"
        return self.metadata[id]
    
    def is_empty(self) -> bool:
        """Checks if the BuilderMetadata is empty."""
        return len(self.metadata) == 0
    
    def set_value(self, id: MetadataID, value: MetadataValue):
        """Sets the Metadata value with given ID."""
        self.metadata[id] = value
        
    def for_each_value(self, func):
        """Iterates over all values in the BuilderMetadata."""
        for id, value in self.metadata.items():
            func(id, value)
            
    def to_json(self) -> str:
        """Converts BuilderMetadata to JSON string."""
        return json.dumps({"m": self.metadata}, indent=2)

    @classmethod
    def from_json(cls, json_str: str) -> 'BuilderMetadata':
        """Creates BuilderMetadata instance from JSON string."""
        data = json.loads(json_str)
        metadata = cls()
        if "m" in data:
            metadata.metadata = data["m"]
        else:
            metadata.metadata = {}
        return metadata

# Singleton implementation with thread safety
class _SingletonHolder:
    _instance = None
    _lock = Lock()

    @classmethod
    def get_instance(cls) -> BuilderMetadata:
        if not cls._instance:
            with cls._lock:
                if not cls._instance:
                    cls._instance = BuilderMetadata()
        return cls._instance

def GlobalBuilderMetadata() -> BuilderMetadata:
    """Returns the global singleton instance of BuilderMetadata."""
    return _SingletonHolder.get_instance()

def Reset():
    """Resets the global BuilderMetadata instance (for testing)."""
    with _SingletonHolder._lock:
        _SingletonHolder._instance = BuilderMetadata()
