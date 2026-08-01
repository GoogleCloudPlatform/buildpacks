import os
import subprocess
from pathlib import Path
from typing import Optional

from google.cloud.buildpacks.gcpbuildpack import Context, InternalError, UserError

class NodeJSBuildPack:
    BUN_LOCK = "bun.lock"
    BUN_LOCKB = "bun.lockb"
    BUN_VERSION_KEY = "version"

    @staticmethod
    def install_bun(ctx: Context, layer: dict, pjs: Optional[dict]) -> None:
        if ctx.is_disabled("nodejs.BunInstaller"):
            ctx.log("BunInstaller capability is disabled. Skipping installation.")
            return

        if capability := ctx.capability("nodejs.BunInstaller"):
            if isinstance(capability, NodeJSBuildPack):
                capability.install_bun(ctx, layer, pjs)
                return

        version = NodeJSBuildPack._detect_bun_version(pjs)
        cached, err = NodeJSBuildPack._install_from_tarball_or_fallback(ctx, version, layer)
        if not cached and err:
            raise InternalError(f"Failed to download Bun v{version} tarball: {err}")

        install_dir = Path(layer['path']) / "bin"
        ctx.setenv("PATH", f"{install_dir}:{os.getenv('PATH')}")
        ctx.set_metadata(layer, NodeJSBuildPack.BUN_VERSION_KEY, version)
        ctx.set_metadata(layer, "stack", ctx.stack_id())

    @staticmethod
    def _detect_bun_version(pjs: Optional[dict]) -> str:
        if pjs is None or (not pjs.get('engines', {}).get('bun') and not pjs.get('packageManager')):
            return NodeJSBuildPack._latest_package_version("bun")

        if engines := pjs.get('engines'):
            bun_version = engines.get('bun')
            if bun_version:
                return bun_version

        package_manager = pjs.get('packageManager', "")
        name, version, err = NodeJSBuildPack.parse_package_manager(package_manager)
        if err or name != "bun":
            raise UserError(f"bun was detected but {name} is set in the packageManager field")

        return version

    @staticmethod
    def _latest_package_version(package_name: str) -> str:
        # TODO: Implement npm version lookup
        return "latest"

    @staticmethod
    def _install_from_tarball_or_fallback(ctx: Context, version: str, layer: dict) -> tuple[bool, Optional[Exception]]:
        try:
            cached = NodeJSBuildPack._install_from_tarball(ctx, version, layer)
            return cached, None
        except Exception as e:
            ctx.log(f"Failed to download Bun v{version} tarball: {e}")
            if not ctx.clear_layer(layer):
                raise InternalError(f"clearing bun layer: failed")

            ctx.log(f"Installing Bun v{version} via script")
            install_cmd = ["bash", "-c", f"curl -fsSL https://bun.sh/install | bash -s bun-v{version} 2>/dev/null"]
            if not NodeJSBuildPack._exec(ctx, install_cmd, env={"BUN_INSTALL": layer['path']}):
                raise InternalError(f"installing bun: {e}")
            
            ctx.set_metadata(layer, NodeJSBuildPack.BUN_VERSION_KEY, version)
            ctx.set_metadata(layer, "stack", ctx.stack_id())
            return False, None

    @staticmethod
    def _install_from_tarball(ctx: Context, version: str, layer: dict) -> bool:
        # TODO(b/520284867): Implement tarball installation
        return False

    @staticmethod
    def parse_package_manager(package_manager_str: str) -> tuple[str, str, Optional[Exception]]:
        parts = package_manager_str.split("@")
        if len(parts) != 2:
            return ("", "", UserError("invalid packageManager format"))
        return (parts[0], parts[1], None)

    @staticmethod
    def _exec(ctx: Context, cmd: list[str], env: Optional[dict] = None) -> bool:
        try:
            result = subprocess.run(cmd, check=True, capture_output=True, text=True, env=env)
            ctx.log(result.stdout.strip())
            return True
        except Exception as e:
            ctx.error(str(e))
            return False
