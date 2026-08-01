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
Implements the legacy GCF Go 1.11 worker buildpack.
The legacy_worker buildpack converts a function into an application and sets up the execution environment.
"""

import os
from pathlib import Path
import textwrap

import gcpbuildpack as gcp
from gcpbuildpack import Context, DetectResult, BuildableLayer, LaunchEnvironment

# Constants
layer_name = "legacy-worker"
gopath_layer_name = "gopath"
app_module = "functions.local/app"
fn_source_dir = "serverless_function_source_code"

# Global variables
google_dirs = [fn_source_dir, ".googlebuild", ".googleconfig"]
worker_tmpl_file = textwrap.dedent("""\
    import (
        "log"
        "net/http"
        userfunction "{{ .Package }}"
    )

    func extraUseOfUserFunction() {
        var handler interface{} = userfunction.{{ .Target }}
    }

    func main() {
        var handler interface{} = userfunction.{{ .Target }}
        http.HandleFunc("/", extraUseOfUserFunction)
        err := http.ListenAndServe(":"+"8080", nil)
        if err != nil {
            log.Fatalf("Error starting the Worker server for Go: %s\n", err)
        }
    }
""")

go_mod_tmpl_file = textwrap.dedent("""\
    // Copyright 2021 Google LLC
    //
    // Licensed under the Apache License, Version 2.0 (the "License");
    // you may not use this file except in compliance with the License.
    // You may obtain a copy of the License at
    //
    //      http://www.apache.org/licenses/LICENSE-2.0
    //
    // Unless required by applicable law or agreed to in writing, software
    // distributed under the License is distributed on an "AS IS" BASIS,
    // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    // See the License for the specific language governing permissions and
    // limitations under the License.

    module functions.local/app

    require fnmod v0.0.0

    replace fnmod => ./serverless_function_source_code
""")

class FnInfo:
    def __init__(self, source: str, target: str, package: str):
        self.source = source
        self.target = target
        self.package = package

def detect_fn(ctx: Context) -> DetectResult:
    if not os.getenv("GOOGLE_RUNTIME") == "go111":
        return DetectResult.OptOut("Only compatible with go111")
    
    function_target = os.getenv(env.FunctionTarget)
    if function_target is not None:
        return DetectResult.OptInEnvSet(env.FunctionTarget)
    
    return DetectResult.OptOutEnvNotSet(env.FunctionTarget)

def build_fn(ctx: Context) -> None:
    try:
        layer = ctx.layer(layer_name, BuildableLayer.LAUNCH_LAYER)
    except Exception as e:
        raise RuntimeError(f"Creating {layer_name} layer failed: {e}") from e
    
    if not ctx.set_functions_env_vars(layer):
        return
    
    ctx.add_web_process(["go", "out", "bin"])

    fn_target = os.getenv(env.FunctionTarget)

    # Move function source code into subdirectory
    try:
        ctx.remove_all(fn_source_dir)
        ctx.mkdir_all(fn_source_dir, 0o755)
    except Exception as e:
        raise RuntimeError(f"Failed to prepare {fn_source_dir}: {e}") from e

    command = f"find . -mindepth 1 -not -name '{fn_source_dir}' -prune -not -name '.google*' -prune -exec mv -t {fn_source_dir} {{}} +"
    if not ctx.exec(["bash", "-c", command], user_timing_attribution=True):
        return

    fn_source = str(Path(ctx.application_root()) / fn_source_dir)
    try:
        pkg_name = extract_package_name_in_dir(ctx, fn_source)
    except Exception as e:
        raise RuntimeError(f"Extracting package name failed: {e}") from e
    
    fn = FnInfo(source=fn_source, target=fn_target, package=pkg_name)

    layer.launch_environment.default("X_GOOGLE_ENTRY_POINT", os.getenv(env.FunctionTarget))
    trigger_type = os.getenv(env.FunctionSignatureType)
    if trigger_type in ("http", ""):
        trigger_type = "HTTP_TRIGGER"
    layer.launch_environment.default("X_GOOGLE_FUNCTION_TRIGGER_TYPE", trigger_type)

    go_mod_path = str(Path(fn.source) / "go.mod")
    try:
        go_mod_exists = ctx.file_exists(go_mod_path)
    except Exception as e:
        raise RuntimeError(f"Checking {go_mod_path} existence failed: {e}") from e
    
    if not go_mod_exists:
        create_main_vendored(ctx, fn)
        return

    is_writable = ctx.is_writable(go_mod_path)
    if not is_writable:
        raise gcp.UserError("go.mod exists but is not writable")
    
    create_main_go_mod(ctx, fn)

def create_main_go_mod(ctx: Context, fn: FnInfo) -> None:
    try:
        layer = ctx.layer(gopath_layer_name, BuildableLayer.BUILD_LAYER)
    except Exception as e:
        raise RuntimeError(f"Creating {gopath_layer_name} layer failed: {e}") from e
    
    layer.build_environment.override("GOPATH", str(layer.path))
    if not ctx.setenv("GOPATH", str(layer.path)):
        return

    fn_mod, fn_package, err = module_and_package_names(ctx, fn)
    if err:
        raise RuntimeError(f"Extracting module and package names failed: {err}")

    fn.package = fn_package

    create_main_go_mod_file(ctx, fn_mod, str(Path(ctx.application_root()) / "go.mod"))
    create_main_go_file(ctx, fn, str(Path(ctx.application_root()) / "main.go"))

def create_main_go_mod_file(ctx: Context, fn_mod: str, go_mod_path: str) -> None:
    try:
        with ctx.create_file(go_mod_path) as f:
            tmpl = textwrap.Template(go_mod_tmpl_file)
            tmpl_subs = {
                "AppModule": app_module,
                "FnModule": fn_mod,
                "FnSource": fn_source_dir
            }
            f.write(tmpl.substitute(**tmpl_subs))
    except Exception as e:
        raise RuntimeError(f"Error creating go.mod for function application: {e}") from e

def module_and_package_names(ctx: Context, fn: FnInfo) -> tuple[str, str, Exception]:
    try:
        result = ctx.exec(
            ["go", "list", "-m"],
            work_dir=fn.source,
            user_attribution=True
        )
    except Exception as e:
        return ("", "", e)
    
    fn_mod = result.stdout.strip()
    fn_package = fn_mod

    package_path = str(Path(ctx.application_root()) / fn.package)
    try:
        if ctx.file_exists(package_path):
            fn_package = f"{fn_mod}/{fn.package}"
    except Exception as e:
        return ("", "", e)
    
    return (fn_mod, fn_package, None)

def create_main_vendored(ctx: Context, fn: FnInfo) -> None:
    try:
        layer = ctx.layer(gopath_layer_name, BuildableLayer.BUILD_LAYER)
    except Exception as e:
        raise RuntimeError(f"Creating {gopath_layer_name} layer failed: {e}") from e
    
    gopath = str(ctx.application_root())
    gopath_src = str(Path(gopath) / "src")
    
    if not ctx.mkdir_all(gopath_src, 0o755):
        return

    layer.build_environment.override(env.Buildable, f"{app_module}/main")
    layer.build_environment.override("GOPATH", gopath)
    layer.build_environment.override("GO111MODULE", "auto")

    if not ctx.setenv("GOPATH", gopath):
        return

    app_path = str(Path(gopath_src) / app_module / "main")
    if not ctx.mkdir_all(app_path, 0o755):
        return

    try:
        ctx.rename(fn.source, str(Path(gopath_src) / fn.package))
    except Exception as e:
        raise RuntimeError(f"Failed to rename directory: {e}") from e
    
    create_main_go_file(ctx, fn, str(Path(app_path) / "main.go"))

def create_main_go_file(ctx: Context, fn: FnInfo, main_path: str) -> None:
    try:
        with ctx.create_file(main_path) as f:
            tmpl = textwrap.Template(worker_tmpl_file)
            f.write(tmpl.substitute(fn.__dict__))
    except Exception as e:
        raise RuntimeError(f"Error creating main.go file: {e}") from e

def extract_package_name_in_dir(ctx: Context, source: str) -> tuple[str, Exception]:
    script_path = str(Path(ctx.buildpack_root()) / "converter" / "get_package" / "main.py")
    cache_dir = ctx.temp_dir("app")

    try:
        result = ctx.exec(
            ["python3", script_path, "-dir", source],
            env={"GOCACHE": str(cache_dir)},
            user_attribution=True
        )
    except Exception as e:
        return ("", e)
    
    return (result.stdout.strip(), None)
