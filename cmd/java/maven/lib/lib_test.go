# Complete refactored code here
"""
Implements java/maven buildpack.
The maven buildpack builds Maven applications.
"""

import os
import shutil
from pathlib import Path
from typing import Any, Dict, List, Optional

import googlecloudplatform.buildpacks.pkg.devmode as devmode
import googlecloudplatform.buildpacks.pkg.env as env
import googlecloudplatform.buildpacks(pkg.fileutil as fileutil)
import googlecloudplatform.buildpacks.pkg.gcpbuildpack as gcp
import googlecloudplatform.buildpacks.pkg.java as java

m2_layer = "m2"

def detect_fn(ctx: gcp.Context) -> Dict[str, Any]:
    """
    Detect function for Maven buildpack.
    
    Args:
        ctx: The build context containing environment and file information.
        
    Returns:
        A dictionary indicating whether the buildpack should be used.
    """
    pom_path = _find_pom_file(ctx)
    if pom_path is not None:
        return {"result": "OptInFileFound", "file": "pom.xml"}
    
    ext_xml_exists = ctx.file_exists(".mvn/extensions.xml")
    if ext_xml_exists:
        return {"result": "OptInFileFound", "file": ".mvn/extensions.xml"}
        
    return {"result": "OptOut", 
            "reason": "none of the following found: pom.xml or .mvn/extensions.xml."}

def build_fn(ctx: gcp.Context) -> None:
    """
    Build function for Maven buildpack.
    
    Args:
        ctx: The build context containing environment and file information.
    """
    m2_cached_repo = _create_m2_layer(ctx)
    _validate_cache(ctx, m2_cached_repo)
    
    home_m2 = os.path.join(ctx.home_dir(), ".m2")
    if os.path.exists(home_m2):
        shutil.rmtree(home_m2)
        
    os.symlink(m2_cached_repo.path, home_m2)
    
    _add_jvm_config(ctx)
    mvn_path = _provision_or_detect_maven(ctx)
    
    command = [mvn_path, "clean", "package", "--batch-mode", "-DskipTests", "-Dhttp.keepAlive=false"]
    pom_path = _find_pom_file(ctx)
    if pom_path:
        command.append(f"-f={pom_path}")
        
    build_args = os.getenv(env.build_args)
    if build_args:
        if "maven.repo.local" in build_args:
            ctx.warn("Detected maven.repo.local property set in GOOGLE_BUILD_ARGS. Maven caching may not work properly.")
        command.extend(build_args.split())
    
    mvn_build_args = os.getenv(java.maven_build_args)
    if mvn_build_args:
        command = [mvn_path] + mvn_build_args.split()
        
    if not ctx.debug() and not devmode.enabled(ctx):
        command.append("--quiet")
        
    _execute_command(ctx, command)
    
    if devmode.enabled(ctx):
        devmode.write_build_script(ctx, m2_cached_repo.path, "~/.m2", command)

def _create_m2_layer(ctx: gcp.Context) -> Any:
    """Creates the Maven repository layer."""
    return ctx.layer(m2_layer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)

def _validate_cache(ctx: gcp.Context, m2_cached_repo: Any) -> None:
    """Validates the cache expiration."""
    if not java.check_cache_expiration(ctx, m2_cached_repo):
        raise ValueError("validating the cache failed")

def _add_jvm_config(ctx: gcp.Context) -> None:
    """
    Workaround for Guice reflection warnings.
    
    Args:
        ctx: The build context containing environment and file information.
    """
    version = os.getenv(env.runtime_version)
    if version == "8" or version.startswith("8."):
        return
        
    config_file = ".mvn/jvm.config"
    if not os.path.exists(config_file):
        os.makedirs(".mvn", exist_ok=True, mode=0o755)
        
        try:
            with open(config_file, "w") as f:
                f.write("--add-opens java.base/java.lang=ALL-UNNAMED")
        except Exception as e:
            ctx.log(f"Could not create {config_file}, reflection warnings may not be disabled: {e}")

def _provision_or_detect_maven(ctx: gcp.Context) -> str:
    """
    Detects or provisions Maven.
    
    Args:
        ctx: The build context containing environment and file information.
        
    Returns:
        Path to the Maven executable.
        
    Raises:
        ValueError: If Maven could not be detected or provisioned.
    """
    if os.path.exists("mvnw"):
        _ensure_unix_line_endings("mvnw")
        return "./mvnw"
        
    if _is_maven_installed(ctx):
        return "mvn"
        
    if ctx.is_disabled(java.maven_installer_capability):
        ctx.log("MavenInstaller capability is disabled. Skipping installation of Maven.")
        return ""
        
    mvn_path = java.install_maven(ctx)
    if not mvn_path:
        raise ValueError("installing Maven failed")
    return mvn_path

def _ensure_unix_line_endings(file_path: str) -> None:
    """
    Ensures the file has Unix line endings.
    
    Args:
        file_path: Path to the file to fix.
        
    Raises:
        ValueError: If fixing line endings failed.
    """
    try:
        fileutil.ensure_unix_line_endings(file_path)
    except Exception as e:
        raise ValueError(f"ensuring unix newline characters failed: {e}")

def _is_maven_installed(ctx: gcp.Context) -> bool:
    """
    Checks if Maven is installed.
    
    Args:
        ctx: The build context containing environment and file information.
        
    Returns:
        True if Maven is installed, False otherwise.
    """
    try:
        result = ctx.exec(["bash", "-c", "command -v mvn || true"])
        return bool(result.stdout.strip())
    except Exception as e:
        raise ValueError(f"checking Maven installation failed: {e}")

def _find_pom_file(ctx: gcp.Context) -> Optional[str]:
    """
    Finds the pom.xml file.
    
    Args:
        ctx: The build context containing environment and file information.
        
    Returns:
        Path to the pom.xml file if found, None otherwise.
    """
    buildable = os.getenv(env.buildable)
    if not buildable:
        return None
        
    pom_path = os.path.join(buildable, "pom.xml")
    if ctx.file_exists(pom_path):
        return pom_path
    return None

def _execute_command(ctx: gcp.Context, command: List[str]) -> None:
    """
    Executes the Maven command.
    
    Args:
        ctx: The build context containing environment and file information.
        command: The command to execute.
        
    Raises:
        ValueError: If command execution failed.
    """
    try:
        result = ctx.exec(command, gcp.WithStdoutTail, gcp.WithUserAttribution)
        if result.exit_code != 0:
            raise ValueError(f"command execution failed with exit code {result.exit_code}")
    except Exception as e:
        raise ValueError(f"error executing command: {e}")
