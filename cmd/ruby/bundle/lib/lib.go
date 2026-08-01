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

"""Implements ruby/bundle buildpack."""

import os
from typing import Dict, Optional

import gcpbuildpack as gcp  # type: ignore
from buildererror import BuilderErrorStatus  # type: ignore
from cache import CacheOptions  # type: ignore
from libcnb import Layer  # type: ignore
from ruby import (  # type: ignore
    BundleConfig,
    BundleLocker,
    BundleInstaller,
    RubyVersionKey,
    BundleLockerCapability,
    BundleInstallerCapability,
    PrepareLockfile,
    InstallAndSymlink,
)
import subprocess

LayerName = "gems"
DependencyHashKey = "dependency_hash"

def detect_fn(ctx: gcp.Context) -> tuple[Optional[gcp.DetectResult], Optional[str]]:
    """Detect function for the buildpack."""
    gemfile_exists = os.path.exists("Gemfile")
    if gemfile_exists:
        return gcp.OptInFileFound("Gemfile"), None
    gems_rb_exists = os.path.exists("gems.rb")
    if gems_rb_exists:
        return gcp.OptInFileFound("gems.rb"), None
    return gcp.OptOut("no Gemfile or gems.rb found"), None

def build_fn(ctx: gcp.Context) -> Optional[str]:
    """Build function for the buildpack."""
    lock_file = ""
    has_gemfile = os.path.exists("Gemfile")
    has_gems_rb = os.path.exists("gems.rb")

    if has_gemfile:
        if has_gems_rb:
            ctx.Warn("Gemfile and gems.rb both exist. Using Gemfile.")
        gemfile_lock_exists = os.path.exists("Gemfile.lock")
        if not gemfile_lock_exists:
            return BuilderErrorStatus.FailedPrecondition(
                "Could not find Gemfile.lock file in your app. Please make sure your bundle is up to date before deploying."
            )
        lock_file = "Gemfile.lock"
    elif has_gems_rb:
        gems_locked_exists = os.path.exists("gems.locked")
        if not gems_locked_exists:
            return BuilderErrorStatus.FailedPrecondition(
                "Could not find gems.locked file in your app. Please make sure your bundle is up to date before deploying."
            )
        lock_file = "gems.locked"

    if os.path.exists(".bundle"):
        try:
            os.remove(".bundle")
        except Exception as e:
            return str(e)

    local_gems_dir = os.path.join(".bundle", "gems")
    local_bin_dir = os.path.join(".bundle", "bin")

    cap = ctx.Capability(ruby.BundleLockerCapability)
    if cap:
        locker = cap  # type: BundleLocker
        if not isinstance(locker, BundleLocker):
            return gcp.InternalError(
                f"capability {ruby.BundleLockerCapability} must implement BundleLocker"
            )
        try:
            locker.Lock(ctx)
        except Exception as e:
            return str(e)
    else:
        try:
            PrepareLockfile(ctx, local_gems_dir, "development test", ["x86_64-linux", "ruby"])
            if os.path.exists(".bundle"):
                os.remove(".bundle")
        except Exception as e:
            return str(e)

    cap = ctx.Capability(ruby.BundleInstallerCapability)
    if cap:
        installer = cap  # type: BundleInstaller
        if not isinstance(installer, BundleInstaller):
            return gcp.InternalError(
                f"capability {ruby.BundleInstallerCapability} must implement BundleInstaller"
            )
        try:
            return installer.Install(ctx)
        except Exception as e:
            return str(e)

    try:
        deps = ctx.Layer(LayerName, gcp.BuildLayer | gcp.CacheLayer | gcp.LaunchLayer)
    except Exception as e:
        return f"creating {LayerName} layer: {e}"

    bundle_output = os.path.join(deps.Path, ".bundle")
    cached, err = check_cache(ctx, deps, CacheOptions.Files(lock_file))
    if err:
        return f"checking cache: {err}"

    if cached:
        ctx.CacheHit(LayerName)
    else:
        ctx.CacheMiss(LayerName)

        b_cfg = BundleConfig(Deployment=True, Frozen=True)
        env = ["NOKOGIRI_USE_SYSTEM_LIBRARIES=1", "MALLOC_ARENA_MAX=2", "LANG=C.utf8"]
        try:
            InstallAndSymlink(
                ctx,
                local_gems_dir,
                local_bin_dir,
                "development test",
                b_cfg,
                env
            )
            if os.path.exists(bundle_output):
                os.remove(bundle_output)
            subprocess.run(["mv", ".bundle", bundle_output], check=True)
        except Exception as e:
            return str(e)

    try:
        os.symlink(bundle_output, ".bundle")
    except Exception as e:
        return str(e)

    return None

def gcp_main(detect: callable, build: callable):
    """Main entrypoint for GCP buildpack."""
    gcp.Main(detect, build)

def check_cache(
    ctx: gcp.Context,
    layer: Layer,
    opts: CacheOptions
) -> tuple[bool, Optional[str]]:
    """Check if cached dependencies exist and match."""
    try:
        result = subprocess.run(["ruby", "-v"], capture_output=True, text=True, check=True)
        current_ruby_version = result.stdout
        opts.Strings.append(current_ruby_version)

        hash_val, cached, err = cache.HashAndCheck(ctx, layer, DependencyHashKey, **opts)
        if err:
            return False, err

        if cached:
            return True, None

        ctx.Log("Installing application dependencies.")
        cache.Add(ctx, layer, DependencyHashKey, hash_val)
        ctx.SetMetadata(layer, RubyVersionKey, current_ruby_version)

        return False, None
    except Exception as e:
        return False, str(e)
