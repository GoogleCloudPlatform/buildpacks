import json
import os
import requests
from typing import Optional
from urllib.parse import urljoin

VERSION_URL = "https://storage.googleapis.com/dart-archive/channels/stable/release/latest/VERSION"

class ReleaseInfo:
    def __init__(self, date: str, version: str, revision: str):
        self.date = date
        self.version = version
        self.revision = revision

def detect_sdk_version() -> tuple[Optional[str], Optional[Exception]]:
    env_version = os.getenv("GOOGLE_RUNTIME_VERSION")
    if env_version:
        return env_version, None
    return fetch_latest_sdk_version()

def fetch_latest_sdk_version() -> tuple[Optional[str], Optional[Exception]]:
    try:
        response = requests.get(VERSION_URL, retries=3)
        response.raise_for_status()
        
        data = json.loads(response.text)
        if not isinstance(data, dict):
            return None, ValueError("Invalid JSON format in response")
            
        version = data.get("version")
        if not version:
            return None, ValueError("Version not found in response")
            
        return version, None
    except requests.exceptions.RequestException as e:
        return None, e

def has_build_runner(directory: str) -> tuple[bool, Optional[Exception]]:
    pubspec_path = os.path.join(directory, "pubspec.yaml")
    
    try:
        with open(pubspec_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        return False, None
    except IOError as e:
        return False, e
    
    try:
        pubspec = json.loads(content)
    except json.JSONDecodeError as e:
        return False, e
    
    dependencies = pubspec.get("dependencies", {})
    dev_dependencies = pubspec.get("dev_dependencies", {})
    
    if "build_runner" in dependencies or "build_runner" in dev_dependencies:
        return True, None
    return False, None
