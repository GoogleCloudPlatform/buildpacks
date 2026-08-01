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

"""Implements nodejs/npm buildpack. The npm buildpack installs dependencies using npm."""

import os
import subprocess
from pathlib import Path

import gcpbuildpack as gcp
import ar
import buildermetadata
import buildermetrics
import cache
import devmode
import env
import firebase.faherror
import nodejs

cache_tag = "prod dependencies"


def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception | None]:
    """Detects if package.json is present."""
    try:
        if ctx.file_exists("package.json"):
            return gcp.opt_in_file_found("package.json"), None
        return gcp.opt_out_file_not_found("package.json"), None
    except Exception as e:
        return None, e


def build_fn(ctx: gcp.Context) -> Exception | None:
    """Builds the npm project by installing dependencies."""
    try:
        buildermetadata.global_builder_metadata().set_value(
            buildermetadata.PackageManager,
            buildermetadata.MetadataValue("npm")
        )
        
        # Layer handling
        ml = ctx.layer("npm_modules", gcp.BuildLayer, gcp.CacheLayer)
        nm = Path(ml.path) / "node_modules"
        
        # Node modules cleanup and metrics
        if (Path(ctx.application_root()) / "node_modules").exists():
            buildermetrics.global_builder_metrics().get_counter(
                buildermetrics.NpmNodeModulesCounterID
            ).increment(1)

        vendor_npm_deps = nodejs.is_using_vendored_dependencies()
        
        # Clean up existing node_modules if not vendoring dependencies
        if not vendor_npm_deps:
            if ctx.remove_all("node_modules") != 0:
                return Exception("Failed to clean up node_modules")

        # Generate NPM config
        if ar.generate_npm_config(ctx) != 0:
            return Exception("Generating Artifact Registry credentials failed")

        # Read and process package.json
        pjs = nodejs.read_package_json_if_exists(ctx.application_root())
        if not pjs:
            return Exception("Failed to read package.json")

        # Upgrade npm if needed
        error = upgrade_npm(ctx, pjs)
        if error:
            vendor_error = ""
            if vendor_npm_deps:
                vendor_error = "Vendored dependencies detected, please remove the npm version from your package.json to avoid installing npm and instead use the bundled npm"
            return Exception(f"{vendor_error} Error: {error}")

        # Ensure lockfile
        lockfile = nodejs.ensure_lockfile(ctx)
        if not lockfile:
            return Exception("Failed to ensure lockfile")

        # Override build script for AppHosting
        pjs, err = nodejs.override_app_hosting_build_script(
            ctx.application_root(),
            nodejs.ApphostingPreprocessedPathForPack
        )
        if err:
            return err

        # Determine build commands and environment
        build_cmds, is_custom_build = nodejs.determine_build_commands(pjs, "npm")
        
        # Determine NODE_ENV
        build_node_env = os.getenv(nodejs.EnvNodeEnv)
        if not build_node_env:
            if len(build_cmds) > 0:
                build_node_env = nodejs.EnvDevelopment
            else:
                build_node_env = nodejs.EnvProduction

        # Handle vendoring vs caching
        if vendor_npm_deps:
            buildermetrics.global_builder_metrics().get_counter(
                buildermetrics.NpmVendorDependenciesCounterID
            ).increment(1)
            
            # Rebuild vendored dependencies
            result = ctx.exec(["npm", "rebuild"], env={"NODE_ENV": build_node_env})
            if result != 0:
                return Exception("Failed to rebuild vendored dependencies")
        else:
            # Check or clear cache
            cached, err = nodejs.check_or_clear_cache(
                ctx,
                ml,
                cache.WithStrings(build_node_env),
                cache.WithFiles("package.json", lockfile)
            )
            if err:
                return f"Checking cache failed: {err}"

            if cached:
                # Restore cached modules and run npm install
                if nodejs.restore_modules(ctx, str(nm), "node_modules") != 0:
                    return Exception("Failed to restore cached modules")
                
                result = ctx.exec(
                    ["npm", "install", "--quiet"],
                    env={"NODE_ENV": build_node_env}
                )
                if result != 0:
                    return Exception("Failed to install dependencies")
            else:
                # Install new dependencies
                ctx.log("Installing application dependencies.")
                install_cmd, err = nodejs.npm_install_command(ctx)
                if err:
                    return f"Getting npm install command failed: {err}"

                cmd = ["npm", install_cmd, "--quiet", "--no-fund", "--no-audit"]
                result = ctx.exec(cmd, env={"NODE_ENV": build_node_env})
                if result != 0:
                    return Exception("Failed to install dependencies")

                if nodejs.save_modules(ctx, "node_modules", str(nm)) != 0:
                    return Exception("Failed to save modules")

        # Check for vulnerabilities
        node_deps = nodejs.read_node_dependencies(ctx.application_root())
        if not node_deps:
            ctx.warn(f"Failed to read node dependencies: {node_deps}")
        else:
            err = nodejs.check_vulnerabilities(ctx, node_deps)
            if err:
                return f"Checking vulnerabilities failed: {err}"

        # Run build commands
        if len(build_cmds) > 0:
            for cmd in build_cmds.split():
                exec_opts = [gcp.WithUserAttribution]
                if nodejs.detect_svelte_kit_auto_adapter(pjs):
                    exec_opts.append(gcp.WithEnv(nodejs.SvelteAdapterEnv))

                result = ctx.exec(cmd.split(), *exec_opts)
                if result != 0:
                    if not is_custom_build:
                        return f"Build command failed: {cmd}"
                    return f"Custom build command failed: {cmd}"

            # Prune dev dependencies if needed
            if should_prune(ctx, pjs):
                result = ctx.exec(["npm", "prune", "--production"])
                if result != 0:
                    return Exception("Failed to prune dev dependencies")

        # Configure environment and entrypoint
        el = ctx.layer("env", gcp.BuildLayer, gcp.LaunchLayer)
        env_path = str(Path(ctx.application_root()) / "node_modules" / ".bin")
        el.shared_environment.prepend("PATH", os.pathsep, env_path)
        el.shared_environment.default("NODE_ENV", nodejs.node_env())

        # Configure entrypoint
        cmd, err = nodejs.default_start_command(ctx, pjs)
        if err:
            return f"Detecting start command failed: {err}"

        dev_sync = env.is_dev_sync()
        if dev_sync is None:
            ctx.warn("Unable to determine dev sync status")
        elif dev_sync:
            cmd, err = nodejs.dev_sync_entrypoint(ctx, pjs, "npm")
            if err:
                return f"Getting dev sync entrypoint failed: {err}"
            ctx.add_web_process(cmd)
            return

        if not devmode.enabled(ctx):
            ctx.add_web_process(cmd)
            return

        # Configure for development mode
        err = devmode.add_file_watcher_process(
            ctx,
            devmode.Config(
                run_cmd=cmd,
                ext=devmode.NodeWatchedExtensions
            )
        )
        if err:
            return f"Adding dev mode file watcher failed: {err}"

        return None

    except Exception as e:
        return e


def should_prune(ctx: gcp.Context, pjs) -> bool:
    """Determines if dev dependencies should be pruned."""
    if nodejs.is_using_vendored_dependencies():
        return False
    if not nodejs.has_dev_dependencies(pjs):
        return False
    if os.getenv(nodejs.EnvNodeEnv, "") != nodejs.EnvProduction:
        ctx.log(f"Retaining devDependencies because NODE_ENV={os.getenv(nodejs.EnvNodeEnv, '')}")
        return False
    if nodejs.skip_pruning_dev_sync(ctx):
        return False
    can_prune, err = nodejs.supports_npm_prune(ctx)
    if not can_prune:
        ctx.warn("Retaining devDependencies because npm prune is not supported")
        return False
    return True


def upgrade_npm(ctx: gcp.Context, pjs) -> Exception | None:
    """Upgrades npm to the requested version."""
    npm_version = nodejs.requested_npm_version(pjs)
    if not npm_version:
        return None

    npm_layer = ctx.layer("npm", gcp.BuildLayer, gcp.LaunchLayer, gcp.CacheLayer)
    meta_version = ctx.get_metadata(npm_layer, "version")
    if meta_version == npm_version:
        ctx.log(f"npm@{npm_version} cache hit, skipping installation.")
        return None

    ctx.clear_layer(npm_layer)
    prefix = f"--prefix={npm_layer.path}"
    package_arg = f"npm@{npm_version}"

    result = ctx.exec(["npm", "install", "-g", prefix, package_arg])
    if result != 0:
        return Exception("Failed to install npm version")

    # Update PATH
    new_path = os.pathsep.join([os.getenv("PATH"), str(Path(npm_layer.path) / "bin")])
    if ctx.setenv("PATH", new_path):
        return None
    else:
        return Exception("Failed to update PATH environment variable")
