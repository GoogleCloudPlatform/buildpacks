import os
import subprocess
from typing import List, Optional

import pkg.appengine as appengine_pkg
import pkg.env as env_pkg
from gcpbuildpack.context import Context
from gcpbuildpack.detection import DetectResult, OptInEnvSet, OptOutEnvNotSet

def detect_fn(ctx: Context) -> tuple[DetectResult, Optional[str]]:
    if env_pkg.is_gae():
        return OptInEnvSet(env_pkg.X_GOOGLE_TARGET_PLATFORM), None
    
    return OptOutEnvNotSet(env_pkg.X_GOOGLE_TARGET_PLATFORM), None

def build_fn(ctx: Context) -> None:
    validate_appengine_apis(ctx)
    appengine_pkg.build(ctx, "php", [])

def validate_appengine_apis(ctx: Context) -> None:
    composer_json_exists = os.path.exists("composer.json")
    if not composer_json_exists:
        return
    
    supports_apis = php.supports_appengine_apis(ctx)
    
    direct_deps = get_direct_deps(ctx)
    if not supports_apis and has_appengine_dependency(direct_deps):
        ctx.warn(
            "There is a direct dependency on App Engine APIs, but they are not enabled in app.yaml (set the app_engine_apis property)"
        )
        return
    
    all_deps = get_all_deps(ctx)
    
    using_apis = has_appengine_dependency(all_deps)
    if supports_apis and not using_apis:
        ctx.warn(
            "App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app's yaml config file."
        )
        return
    
    if not supports_apis and using_apis:
        ctx.warn(
            "There is an indirect dependency on App Engine APIs, but they are not enabled in app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property."
        )

def has_appengine_dependency(deps: List[str]) -> bool:
    for dep in deps:
        if dep.startswith("google/appengine-php-sdk"):
            return True
    return False

def get_all_deps(ctx: Context) -> List[str]:
    result = subprocess.run(
        ["composer", "show", "-N"],
        capture_output=True,
        text=True,
        check=True,
        env=ctx.user_env
    )
    return result.stdout.strip().split('\n')

def get_direct_deps(ctx: Context) -> List[str]:
    result = subprocess.run(
        ["composer", "show", "--direct", "-N"],
        capture_output=True,
        text=True,
        check=True,
        env=ctx.user_env
    )
    return result.stdout.strip().split('\n')
