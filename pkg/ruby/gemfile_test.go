import os
import re
import semver
from typing import Optional
import libcnb

GEMS_INSTALLER_CAPABILITY = "ruby.GemsInstaller"
BUNDLE_LOCKER_CAPABILITY = "ruby.BundleLocker"
BUNDLE_INSTALLER_CAPABILITY = "ruby.BundleInstaller"

class GemsInstaller:
    def install(self, ctx: GCPContext, layer: libcnb.Layer) -> None:
        pass

class BundleLocker:
    async def lock(self, ctx: GCPContext) -> None:
        pass

class MakerBundleLocker(BundleLocker):
    async def lock(self, ctx: GCPContext) -> None:
        local_gems_dir = os.path.join(".bundle", "gems")
        await prepare_lockfile(ctx, local_gems_dir, "development test", ["x86_64-linux", "ruby"])

class BundleInstaller:
    async def install(self, ctx: GCPContext) -> None:
        pass

class MakerBundleInstaller(BundleInstaller):
    async def install(self, ctx: GCPContext) -> None:
        local_gems_dir = os.path.join(".bundle", "gems")
        local_bin_dir = os.path.join(".bundle", "bin")
        
        env = ["NOKOGIRI_USE_SYSTEM_LIBRARIES=1", "MALLOC_ARENA_MAX=2", "LANG=C.utf8"]
        await install_and_symlink(ctx, local_gems_dir, local_bin_dir, "development test", BundleConfig(), env)

class BundleConfig:
    def __init__(self):
        self.force_ruby_platform = False
        self.deployment = False
        self.frozen = False

async def prepare_lockfile(ctx: GCPContext, gems_dir: str, without: str, platforms: list[str]) -> None:
    if without:
        await ctx.exec(["bundle", "config", "--local", "without", without])
    
    if gems_dir:
        await ctx.exec(["bundle", "config", "--local", "path", gems_dir])
    
    for platform in platforms:
        await ctx.exec(["bundle", "lock", "--add-platform", platform])

async def install_and_symlink(ctx: GCPContext, gems_dir: str, bin_dir: str, without: str, cfg: BundleConfig, env: list[str]) -> None:
    if without:
        await ctx.exec(["bundle", "config", "--local", "without", without])
    
    if gems_dir:
        await ctx.exec(["bundle", "config", "--local", "path", gems_dir])
    
    if cfg.force_ruby_platform:
        await ctx.exec(["bundle", "config", "--local", "force_ruby_platform", "true"])
    
    if cfg.deployment:
        await ctx.exec(["bundle", "config", "--local", "deployment", "true"])
    
    if cfg.frozen:
        await ctx.exec(["bundle", "config", "--local", "frozen", "true"])

    install_cmd = ["bundle", "install"]
    exec_opts = []
    if env:
        exec_opts.append(("env", env))
    
    await ctx.exec(install_cmd, *exec_opts)

async def symlink_bin(ctx: GCPContext, gems_dir: str, bin_dir: str) -> None:
    glob_pattern = os.path.join(gems_dir, "ruby", "*", "bin")
    found_bin_dirs = await ctx.glob(glob_pattern)
    
    if len(found_bin_dirs) > 1:
        raise ValueError(f"unexpected multiple gem bin dirs: {found_bin_dirs}")
    elif found_bin_dirs:
        if await ctx.remove_all(bin_dir):
            rel_target = os.path.relpath(found_bin_dirs[0], os.path.dirname(bin_dir))
            await ctx.symlink(rel_target, bin_dir)
