# Complete refactored code here
"""
Implements nodejs/firebasenextjs buildpack.
The nodejs/firebasenextjs buildpack does some prep work for nextjs and overwrites the build script.
"""

import os
import json
from pathlib import Path

import semver
import packaging.version as version_utils

import gcpbuildpack as gcp
import buildermetadata
import firebase.apphostingschema as apphosting_schema
import firebase.faherror as fah_error
import firebase.util as util
import nodejs as npm

# Framework constants
FRAMEWORK_VERSION = "FRAMEWORK_VERSION"
MIN_NEXT_VERSION = semver.Version.parse("13.0.0")

class NodeDeps:
    def __init__(self, package_json=None, lock_file_path="", npm_modules_path=""):
        self.package_json = package_json or {}
        self.lock_file_path = lock_file_path
        self.npm_modules_path = npm_modules_path

def read_node_dependencies(ctx, app_dir):
    """
    Reads node.js dependencies from package.json and lock files.
    """
    package_json_path = os.path.join(app_dir, "package.json")
    
    if not os.path.exists(package_json_path):
        return None
    
    with open(package_json_path) as f:
        package_json = json.load(f)
        
    lock_file_paths = [
        os.path.join(app_dir, "package-lock.json"),
        os.path.join(app_dir, "pnpm-lock.yaml"),
        os.path.join(app_dir, "yarn.lock")
    ]
    
    for path in lock_file_paths:
        if os.path.exists(path):
            return NodeDeps(package_json=package_json, lock_file_path=path)
            
    return NodeDeps(package_json=package_json)

def detect_fn(ctx):
    """
    Detects whether this buildpack should be applied.
    """
    app_dir = util.get_application_directory(ctx)
    
    if not os.environ.get("X_GOOGLE_TARGET_PLATFORM") == "fah":
        return (False, "not a firebase apphosting application")
        
    node_deps = read_node_dependencies(ctx, app_dir)
    if not node_deps:
        ctx.warn("Error reading node dependencies")
        return (False, None)
        
    # Check for apphosting build scripts
    apphosting_schema = apphosting_schema.read_and_validate_from_file(npm.APPHOSTING_PREPROCESSED_PATH_FOR_PACK)
    if npm.has_apphosting_package_or_yaml_build(node_deps.package_json.get("dependencies", {}), apphosting_schema):
        return (False, "apphosting build script found")
        
    # Check for Next.js config files
    supported_configs = ["next.config.js", "next.config.mjs", "next.config.ts"]
    for config in supported_configs:
        if os.path.exists(os.path.join(app_dir, config)):
            return (True, f"Next.js config file {config} found")
            
    # Check package.json dependencies
    next_version = npm.get_version(node_deps, "next")
    if next_version:
        return (True, "Next.js dependency found")
        
    return (False, "Next.js config or dependency not found")

def build_fn(ctx):
    """
    Builds the application with Next.js specific configurations.
    """
    app_dir = util.get_application_directory(ctx)
    node_deps = read_node_dependencies(ctx, app_dir)
    
    if not node_deps.lock_file_path:
        ctx.error(f"Missing lock file in directory: {app_dir}")
        return
        
    # Validate Next.js version
    next_version = npm.get_version(node_deps, "next")
    if not next_version:
        next_version = node_deps.package_json.get("dependencies", {}).get("next", "")
    
    validate_version(ctx, next_version)
        
    # Check for existing adapter
    adapter_dep = node_deps.package_json.get("dependencies", {}).get("@apphosting/adapter-nextjs")
    if adapter_dep:
        ctx.log(f"*** Using existing @apphosting/adapter-nextjs@{adapter_dep} ***")
        return
        
    # Install Next.js build adapter
    npm.install_next_js_build_adapter(ctx, os.path.join(app_dir, "node_modules"))
    
    # Set environment variables for build
    env = {
        FRAMEWORK_VERSION: next_version
    }
    ctx.set_build_environment(env)
    
    # Update builder metadata
    buildermetadata.global_builder_metadata().set_value(buildermetadata.FrameworkName, "nextjs")
    buildermetadata.global_builder_metadata().set_value(buildermetadata.FrameworkVersion, next_version)
    
def validate_version(ctx, dep_version):
    """
    Validates the Next.js version against minimum requirements.
    """
    try:
        parsed_version = semver.Version.parse(dep_version)
    except ValueError:
        ctx.warn(f"Unrecognized version of next: {dep_version}")
        ctx.warn("Consider updating your next dependencies to >=13.0.0")
        return
        
    if parsed_version < MIN_NEXT_VERSION:
        ctx.warn(f"Unsupported version of next: {dep_version}")
        ctx.warn("Update the next dependencies to >=13.0.0")
        raise fah_error.UnsupportedFrameworkVersionError("next", dep_version)
