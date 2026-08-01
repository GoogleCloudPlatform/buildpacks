# Complete refactored code here
import json
import logging
import os
import glob

from pkg import env
from pkg import gcpbuildpack as gcp
from pkg.runtime import runtime

logger = logging.getLogger(__name__)

JAVA_LAYER = "java"

DEFAULT_FEATURE_VERSION = {
    "google": "11",
    "google.gae.18": "11",
    "google.18": "11",
    "google.gae.22": "21",
    "google.min.22": "21",
    "google.22": "21",
    "google.24": "25",
    "google.24.full": "25",
}

def detect_fn(ctx: gcp.Context) -> (gcp.DetectResult, Exception):
    if runtime.check_override("java"):
        return None, None  # Opt out
    
    files = [
        "pom.xml",
        ".mvn/extensions.xml",
        "build.gradle",
        "build.gradle.kts",
        "settings.gradle.kts",
        "settings.gradle",
        "META-INF/MANIFEST.MF"
    ]
    
    for file in files:
        logger.info(f"Checking for file: {file}")
        if ctx.file_exists(file):
            return gcp.OptInFileFound(file), None
    
    java_files = glob.glob("*.java")
    if len(java_files) > 0:
        return gcp.OptIn("found .java files"), None
    
    jar_files = glob.glob("*.jar")
    if len(jar_files) > 0:
        return gcp.OptIn("found .jar files"), None
    
    missing_files = ", ".join(files)
    return gcp.OptOut(f"none of the following found: {missing_files}, *.java, *.jar"), None

def build_fn(ctx: gcp.Context) -> Exception:
    stack_id = ctx.stack_id()
    feature_version = stack_to_version(stack_id)
    
    runtime_env = os.getenv(env.RUNTIME_VERSION)
    if runtime_env:
        feature_version = runtime_env
        logger.info(f"Using requested runtime feature version: {feature_version}")
    else:
        logger.info(f"Using latest Java {feature_version} runtime version. You can specify a different version with {env.RUNTIME_VERSION}: https://github.com/GoogleCloudPlatform/buildpacks#configuration")
    
    layer, err = ctx.create_layer(JAVA_LAYER, gcp.BuildLayer | gcp.CacheLayer | gcp.LaunchLayerUnlessSkipRuntimeLaunch)
    if err:
        return Exception(f"Creating {JAVA_LAYER} layer failed: {err}")
    
    jdk_runtime = runtime.OPEN_JDK
    if feature_version.startswith("21"):
        jdk_runtime = runtime.CANONICAL_JDK
    
    _, err = runtime.install_tarball_if_not_cached(ctx, jdk_runtime, feature_version, layer)
    return err

def stack_to_version(stack_id: str) -> str:
    feature_version = "21"
    if stack_id in DEFAULT_FEATURE_VERSION:
        feature_version = DEFAULT_FEATURE_VERSION[stack_id]
    return feature_version

def parse_version_json(json_str: str) -> (dict, Exception):
    try:
        releases = json.loads(json_str)
        if not releases:
            return None, Exception("empty list of releases")
        return releases[0], None
    except json.JSONDecodeError as e:
        return None, Exception(f"parsing JSON response {json_str}: {e}")

def extract_release(release: dict) -> (str, str, Exception):
    binaries = release.get("binaries", [])
    if not binaries:
        return "", "", Exception(f"no binaries in given release {release['version_data']['semver']}")
    
    for binary in binaries:
        if (binary["image_type"] == "jdk" and 
            binary["os"] == "linux" and 
            binary["architecture"] == "x64"):
            return release["version_data"]["semver"], binary["package"]["link"], None
    
    return "", "", Exception(f"jdk/linux/x64 binary not found in release {release['version_data']['semver']}")
