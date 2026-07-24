import json
import os
import requests
from typing import Optional, Tuple
from urllib.parse import urljoin

FLUTTER_VERSION_URL = "https://storage.googleapis.com/flutter_infra_release/releases/releases_linux.json"

class ReleaseDetail:
    def __init__(self, hash: str, channel: str, version: str, dart_sdk_version: str,
                 dart_sdk_arch: str, release_date: str, archive: str, sha256: str):
        self.hash = hash
        self.channel = channel
        self.version = version
        self.dart_sdk_version = dart_sdk_version
        self.dart_sdk_arch = dart_sdk_arch
        self.release_date = release_date
        self.archive = archive
        self.sha256 = sha256

class FlutterReleaseInfo:
    def __init__(self, base_url: str, current_release: dict, releases: list[ReleaseDetail]):
        self.base_url = base_url
        self.current_release = current_release
        self.releases = releases

def detect_flutter_sdk_archive() -> Tuple[Optional[str], Optional[str], Optional[Exception]]:
    env_version = os.getenv("GOOGLE_RUNTIME_VERSION")
    if env_version:
        detail, err = fetch_specific_sdk_archive(env_version)
        return (detail.version, detail.archive, err) if detail else (None, None, err)
    
    detail, err = fetch_latest_sdk_archive()
    return (detail.version, detail.archive, err) if detail else (None, None, err)

def download_manifest() -> Tuple[Optional[FlutterReleaseInfo], Optional[Exception]]:
    try:
        response = requests.get(FLUTTER_VERSION_URL, retries=3)
        response.raise_for_status()
        
        data = json.loads(response.text)
        if not isinstance(data, dict):
            return None, ValueError("Invalid JSON format in response")
            
        current_release = {
            "beta": data["current_release"].get("beta", ""),
            "dev": data["current_release"].get("dev", ""),
            "stable": data["current_release"].get("stable", "")
        }
        
        releases = []
        for release_data in data.get("releases", []):
            release = ReleaseDetail(
                hash=release_data.get("hash", ""),
                channel=release_data.get("channel", ""),
                version=release_data.get("version", ""),
                dart_sdk_version=release_data.get("dart_sdk_version", ""),
                dart_sdk_arch=release_data.get("dart_sdk_arch", ""),
                release_date=release_data.get("release_date", ""),
                archive=release_data.get("archive", ""),
                sha256=release_data.get("sha256", "")
            )
            releases.append(release)
            
        return FlutterReleaseInfo(data["base_url"], current_release, releases), None
    except requests.exceptions.RequestException as e:
        return None, e

def fetch_specific_sdk_archive(version: str) -> Tuple[Optional[ReleaseDetail], Optional[Exception]]:
    info, err = download_manifest()
    if err or not info:
        return None, err
    
    for release in info.releases:
        if release.version == version:
            return release, None
    return None, ValueError(f"Version {version} not found")

def fetch_latest_sdk_archive() -> Tuple[Optional[ReleaseDetail], Optional[Exception]]:
    info, err = download_manifest()
    if err or not info:
        return None, err
    
    stable_hash = info.current_release.get("stable", "")
    for release in info.releases:
        if release.hash == stable_hash:
            return release, None
    return None, ValueError("Stable version not found")

def is_flutter(directory: str) -> Tuple[bool, Optional[Exception]]:
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
    if "flutter" in dependencies:
        return True, None
    return False, None

def get_pubspec(directory: str) -> Tuple[Optional[dict], Optional[Exception]]:
    pubspec_path = os.path.join(directory, "pubspec.yaml")
    
    try:
        with open(pubspec_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        return {}, None
    except IOError as e:
        return None, e
    
    try:
        pubspec = json.loads(content)
    except json.JSONDecodeError as e:
        return None, e
    
    buildpack = pubspec.get("buildpack", {})
    if buildpack:
        if not buildpack.get("server"):
            buildpack["server"] = "server"
        if not buildpack.get("static"):
            buildpack["static"] = "static"
        pubspec["buildpack"] = buildpack
    return pubspec, None
