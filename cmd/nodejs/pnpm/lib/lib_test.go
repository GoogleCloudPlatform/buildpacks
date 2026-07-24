#!/usr/bin/env python3.13

# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Implements nodejs/pnpm buildpack. Installs dependencies using pnpm."""

import os
import subprocess
from pathlib import Path

import buildermetadata
import env as gcp_env
import faherror
import gcpbuildpack as gcp
import nodejs


def detect(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """Detects if the buildpack should be applied."""
    try:
        package_json_exists = ctx.file_exists("package.json")
    except Exception as err:
        return None, err

    if not package_json_exists:
        return gcp.OptOutFileNotFound("package.json"), None

    try:
        pnpm_lock_exists = ctx.file_exists(nodejs.PNPM_LOCK)
    except Exception as err:
        return None, err

    if pnpm_lock_exists:
        return gcp.OptIn("found pnpm-lock.yaml and package.json"), None

    if nodejs.is_package_manager_configured("pnpm"):
        return gcp.OptIn("package.json found and GOOGLE_PACKAGE_MANAGER=pnpm"), None

    return gcp.OptOut(
        "pnpm-lock.yaml not found and GOOGLE_PACKAGE_MANAGER is not set to pnpm"
    ), None


def build(ctx: gcp.Context) -> Exception:
    """Builds the application using pnpm."""
    try:
        buildermetadata.GlobalBuilderMetadata().set_value(
            buildermetadata.PackageManager, "pnpm"
        )
    except Exception as err:
        return err

    ctx.setenv("PNPM_CONFIG_VERIFY_DEPS_BEFORE_RUN", "false")

    try:
        pjs = nodejs.read_package_json_if_exists(ctx.application_root())
    except Exception as err:
        return err

    if install_pnpm(ctx, pjs):
        return None

    if pnpm_install_modules(ctx, pjs):
        return None

    try:
        el = ctx.layer("env", gcp.BuildLayer, gcp.LaunchLayer)
    except Exception as err:
        return gcp.InternalError(f"creating layer: {err}")

    bin_path = os.path.join(
        ctx.application_root(), "node_modules", ".bin"
    )
    el.shared_environment.prepend("PATH", os.pathsep, bin_path)
    el.shared_environment.default("NODE_ENV", nodejs.node_env())
    el.shared_environment.default("PNPM_CONFIG_VERIFY_DEPS_BEFORE_RUN", "false")

    try:
        node_deps = nodejs.read_node_dependencies(ctx, ctx.application_root())
    except Exception as err:
        ctx.warn(f"Failed to read node dependencies: {err}")
    else:
        if check_vulnerabilities(ctx, node_deps):
            return None

    entrypoint = nodejs.entrypoint(ctx, "pnpm")
    if not entrypoint:
        return gcp.InternalError("failed to determine entrypoint")

    dev_sync = gcp_env.is_dev_sync()
    if dev_sync is None:
        ctx.warn("Unable to determine dev sync status")
    elif dev_sync:
        entrypoint = nodejs.dev_sync_entrypoint(ctx, pjs, "pnpm")
        if not entrypoint:
            return gcp.InternalError(
                "failed to get dev sync entrypoint"
            )
        ctx.add_web_process(entrypoint)
        return None

    ctx.add_web_process(entrypoint)
    return None


def install_pnpm(ctx: gcp.Context, pjs: nodejs.PackageJSON) -> Exception:
    """Installs pnpm if needed."""
    try:
        layer = ctx.layer(nodejs.PNPM_LAYER, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
    except Exception as err:
        return gcp.InternalError(f"creating {nodejs.PNPM_LAYER} layer: {err}")

    return nodejs.install_pnpm(ctx, layer, pjs)


def pnpm_install_modules(ctx: gcp.Context, pjs: nodejs.PackageJSON) -> Exception:
    """Installs pnpm modules."""
    try:
        pjs = nodejs.override_app_hosting_build_script(
            ctx, nodejs.ApphostingPreprocessedPathForPack
        )
    except Exception as err:
        return err

    build_cmds, _ = nodejs.determine_build_commands(pjs, "pnpm")
    build_node_env = os.getenv(nodejs.EnvNodeEnv)
    if not build_node_env:
        if len(build_cmds) > 0:
            build_node_env = nodejs.EnvDevelopment
        else:
            build_node_env = nodejs.EnvProduction

    cmd = ["pnpm", "install"]
    if build_node_env == nodejs.EnvProduction:
        cmd.append("--prod")

    dev_sync, _ = gcp_env.is_dev_sync()
    if dev_sync:
        cmd.append("--no-frozen-lockfile")

    try:
        ctx.exec(cmd, with_user_attribution=True, env={"CI": "true", "NODE_ENV": build_node_env})
    except Exception as err:
        return gcp.UserError(f"installing pnpm dependencies: {err}")

    if len(build_cmds) > 0:
        for cmd in build_cmds:
            split_cmd = cmd.split()
            try:
                ctx.exec(split_cmd, with_user_attribution=True)
            except Exception as err:
                fah_cmd = os.getenv(nodejs.AppHostingBuildEnv)
                if fah_cmd:
                    return gcp.UserError(f"Failed framework build: {err}")
                if nodejs.has_app_hosting_package_build(pjs):
                    return gcp.UserError(
                        f"Failed framework build with script {pjs.scripts[nodejs.ScriptApphostingBuild]}: {err}"
                    )
                return err

    if should_prune_pnpm_bun(ctx, pjs, build_node_env, bool(build_node_env)):
        cmd = ["pnpm", "prune", "--prod"]
        try:
            ctx.exec(cmd, with_user_attribution=True, env={"CI": "true"})
        except Exception as err:
            return gcp.UserError(f"pruning devDependencies: {err}")

    return None


def should_prune_pnpm_bun(
    ctx: gcp.Context,
    pjs: nodejs.PackageJSON,
    build_node_env: str,
    node_env_present: bool
) -> bool:
    """Determines if dev dependencies should be pruned."""
    return nodejs.should_prune_pnpm_bun(ctx, pjs, build_node_env, node_env_present)
