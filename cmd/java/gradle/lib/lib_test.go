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
Implements java/gradle buildpack.
The gradle buildpack builds Gradle applications.
"""

import os
import subprocess
from pathlib import Path

import gcpbuildpack as gcp  # type: ignore
import devmode  # type: ignore
import env  # type: ignore
import fileutil  # type: ignore
import java  # type: ignore
import runtime  # type: ignore
import tooling  # type: ignore

const = {
    "gradle_distro_url": "https://services.gradle.org/distributions/gradle-%s-bin.zip",
    "gradle_layer": "gradle",
    "cache_layer": "cache",
    "version_key": "version"
}

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """
    Detect function for gradle buildpack.
    Checks for presence of gradle files and returns detection result.
    """
    files = ["build.gradle", "build.gradle.kts", "settings.gradle.kts", "settings.gradle"]
    
    for f in files:
        if ctx.file_exists(f):
            return gcp.OptInFileFound(f), None
    
    message = f"none of the following found: {', '.join(files)}"
    return gcp.OptOut(message), None

def build_fn(ctx: gcp.Context) -> Exception:
    """
    Build function for gradle buildpack.
    Handles gradle setup, installation and execution.
    """
    cache_layer_name = const["cache_layer"]
    gradle_cached_repo, err = ctx.layer(cache_layer_name)
    if err:
        return ValueError(f"creating {cache_layer_name} layer: {err}")
    
    if java.check_cache_expiration(ctx, gradle_cached_repo):
        return ValueError("validating the cache failed")
    
    home_gradle = Path(ctx.home_dir()) / ".gradle"
    try:
        if home_gradle.exists():
            ctx.remove_all(home_gradle)
        
        ctx.symlink(gradle_cached_repo.path, home_gradle)
    except Exception as e:
        return e
    
    gradle, err = provision_or_detect_gradle(ctx)
    if err:
        return err
    
    command = [gradle, "clean", "assemble", "-x", "test", "--build-cache"]
    
    build_args = os.getenv(env.BUILD_ARGS)
    if build_args and "project-cache-dir" in build_args:
        ctx.warn("Detected project-cache-dir property set in GOOGLE_BUILD_ARGS. Dependency caching may not work properly.")
    if build_args:
        command += build_args.split()
    
    gradle_build_args = os.getenv(java.GRADLE_BUILD_ARGS)
    if gradle_build_args:
        command = [gradle] + gradle_build_args.split()
    
    if not ctx.debug() and not devmode.enabled(ctx):
        command.append("--quiet")
    
    try:
        result, err = ctx.exec(command, with_user_attribution=True)
        if err:
            return err
    except Exception as e:
        return e
    
    if devmode.enabled(ctx):
        gradle_cached_repo_path = str(gradle_cached_repo.path)
        home_gradle_str = str(home_gradle)
        devmode.write_build_script(ctx, gradle_cached_repo_path, home_gradle_str, command)
    
    return None

def provision_or_detect_gradle(ctx: gcp.Context) -> tuple[str, Exception]:
    """
    Provisions or detects the gradle executable.
    Checks for gradlew, existing gradle installation, and installs if needed.
    """
    gradlew_path = Path("gradlew")
    if gradlew_path.exists():
        try:
            fileutil.ensure_unix_line_endings(gradlew_path)
        except Exception as e:
            return "", ValueError(f"ensuring unix newline characters: {e}")
        return "./gradlew", None
    
    is_installed, err = gradle_installed(ctx)
    if err:
        return "", err
    if is_installed:
        return "gradle", None
    
    if ctx.is_disabled(java.GradleInstallerCapability):
        ctx.log("GradleInstaller capability is disabled. Skipping installation of Gradle.")
        return "", None
    
    gradle_path, err = install_gradle(ctx)
    if err:
        return "", ValueError(f"installing Gradle: {err}")
    
    return gradle_path, None

def gradle_installed(ctx: gcp.Context) -> tuple[bool, Exception]:
    """
    Checks if gradle is installed.
    """
    try:
        result, _ = ctx.exec(["bash", "-c", "command -v gradle || true"])
        return bool(result.stdout), None
    except Exception as e:
        return False, e

def install_gradle(ctx: gcp.Context) -> tuple[str, Exception]:
    """
    Installs gradle and returns the path of the gradle binary.
    Handles layer management, version resolution, downloading and extraction.
    """
    gradle_layer_name = const["gradle_layer"]
    try:
        gradle_layer, err = ctx.layer(gradle_layer_name)
        if err:
            return "", ValueError(f"creating {gradle_layer_name} layer: {err}")
    except Exception as e:
        return "", e
    
    meta_version = ctx.get_metadata(gradle_layer, const["version_key"])
    
    try:
        gradle_version, err = tooling.resolve_tool_version(
            "java", "gradle", os.getenv(env.RUNTIME_VERSION), runtime.os_for_stack(ctx)
        )
        if err or not gradle_version:
            ctx.warn(f"Could not resolve pinned gradle version, falling back to latest: {err}")
            gradle_version, err = java.get_latest_gradle_version()
            if err:
                return "", ValueError(f"getting latest gradle version: {err}")
    except Exception as e:
        return "", e
    
    if gradle_version == meta_version:
        ctx.cache_hit(gradle_layer_name)
        ctx.log("Gradle cache hit, skipping installation.")
        return str(Path(gradle_layer.path) / "bin" / "gradle"), None
    
    ctx.cache_miss(gradle_layer_name)
    try:
        ctx.clear_layer(gradle_layer)
    except Exception as e:
        return "", ValueError(f"clearing layer {gradle_layer.name}: {e}")
    
    download_url = const["gradle_distro_url"] % gradle_version
    try:
        status_code, err = ctx.http_status(download_url)
        if err or status_code != 200:
            return "", ValueError(
                f"Gradle version {gradle_version} does not exist at {download_url} (status {status_code})"
            )
    except Exception as e:
        return "", e
    
    tmp_dir = Path("/tmp")
    gradle_zip = tmp_dir / "gradle.zip"
    
    try:
        curl_cmd = f"curl --fail --show-error --silent --location --retry 3 {download_url} --output {gradle_zip}"
        result, err = ctx.exec(["bash", "-c", curl_cmd])
        if err:
            return "", err
        
        unzip_cmd = f"unzip -q {gradle_zip} -d {tmp_dir}"
        _, err = ctx.exec(["bash", "-c", unzip_cmd])
        if err:
            return "", err
        
        extracted_path = tmp_dir / f"gradle-{gradle_version}"
        
        install_cmd = f"mv {extracted_path}/* {gradle_layer.path}"
        _, err = ctx.exec(["bash", "-c", install_cmd])
        if err:
            return "", err
        
        ctx.set_metadata(gradle_layer, const["version_key"], gradle_version)
        
        gradle_bin = Path(gradle_layer.path) / "bin" / "gradle"
        return str(gradle_bin), None
    except Exception as e:
        return "", e
    finally:
        if gradle_zip.exists():
            ctx.remove_all(gradle_zip)
        extracted_path = tmp_dir / f"gradle-{gradle_version}"
        if extracted_path.exists():
            ctx.remove_all(extracted_path)
