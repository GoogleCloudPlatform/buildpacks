import os
import logging
from pathlib import Path
from typing import Optional

from pkg.appstart import Config as AppStartConfig, EntrypointGenerator
from pkg.gcpbuildpack import Context, Layer, LaunchLayer


def get_config(ctx: Context, runtime: str, eg: EntrypointGenerator) -> tuple[AppStartConfig, Optional[Exception]]:
    config = AppStartConfig()
    
    if os.getenv(env.runtime):
        ctx.log(f"Using {env.runtime}: {os.getenv(env.runtime)}")
        config.runtime = os.getenv(env.runtime)
    else:
        ctx.log(f"Using runtime: {runtime}")
        config.runtime = runtime

    entrypoint, err = eg(ctx)
    if err:
        return None, Exception(f"getting entrypoint: {err}")

    config.entrypoint = entrypoint
    ctx.log(f"Using config {config}")
    return config, None


def assert_framework_injection_allowed() -> Optional[Exception]:
    should_skip, err = is_skip_framework_injection_enabled()
    if err:
        return err

    if should_skip:
        return Exception("Functions Framework must be set as a dependency when skipping automatic framework injection has been enabled via GOOGLE_SKIP_FRAMEWORK_INJECTION")

    return None


def build(ctx: Context, runtime: str, eg: EntrypointGenerator) -> Optional[Exception]:
    layer, err = ctx.create_layer("serve", LaunchLayer())
    if err:
        return Exception(f"creating layer: {err}")

    bin_dir = Path(layer.path) / "bin"
    try:
        bin_dir.mkdir(parents=True, exist_ok=True)
    except OSError as e:
        return e

    try:
        os.symlink("/usr/bin/serve2", str(bin_dir / "serve"))
    except OSError as e:
        return e

    config, err = get_config(ctx, runtime, eg)
    if err:
        return Exception(f"building config: {err}")

    if not config.write(ctx):
        return Exception("writing config failed")

    ctx.add_web_process(["pid1"])
    return None
