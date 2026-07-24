from dataclasses import dataclass
import json
from pathlib import Path
from typing import Optional

@dataclass
class NxBuild:
    executor: str

@dataclass
class NxTargets:
    build: NxBuild

@dataclass
class NxJSON:
    default_project: str
    nx_cloud_access_token: str

@dataclass
class NxProjectJSON:
    name: str
    project_type: str
    prefix: str
    source_root: str
    targets: NxTargets

def read_nx_jsonIfExists(directory: str = "") -> Optional[NxJSON]:
    path = Path(directory) / "nx.json"
    try:
        content = path.read_text()
        return json.loads(content, cls=NxJSON.from_dict)
    except FileNotFoundError:
        return None
    except Exception as e:
        raise RuntimeError(f"Error reading nx.json: {e}") from e

def read_nx_projectJsonIfExists(directory: str = "") -> Optional[NxProjectJSON]:
    path = Path(directory) / "project.json"
    try:
        content = path.read_text()
        return json.loads(content, cls=NxProjectJSON.from_dict)
    except FileNotFoundError:
        return None
    except Exception as e:
        raise RuntimeError(f"Error reading project.json: {e}") from e
