import json
from pathlib import Path
import requests
from typing import Optional, Dict, Any

class NodeJSRegistry:
    @staticmethod
    def latest_package_version(package_name: str) -> str:
        # Implementation logic here
        return "1.2.3"  # Placeholder

    @staticmethod
    def resolve_package_version(package_name: str, version_constraint: str) -> str:
        # Implementation logic here
        return "1.2.3"  # Placeholder

class YarnTags:
    def __init__(self):
        self.latest = {"stable": "", "latest": ""}
        self.tags = []

def fetch_yarn_tags() -> list[str]:
    try:
        response = requests.get("https://repo.yarnpkg.com/tags")
        response.raise_for_status()
        data = json.loads(response.text, cls=YarnTags.from_dict)
        return data.tags
    except Exception as e:
        raise RuntimeError(f"Error fetching Yarn tags: {e}") from e

def fetch_package_metadata(package_name: str) -> Dict[str, Any]:
    try:
        response = requests.get(
            f"https://registry.npmjs.org/{package_name}",
            headers={"Accept": "application/vnd.npm.install-v1+json"}
        )
        response.raise_for_status()
        return json.loads(response.text)
    except Exception as e:
        raise RuntimeError(f"Error fetching package metadata: {e}") from e
