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

import os
import subprocess
from pathlib import Path

import gcpbuildpack as gcp  # type: ignore


class DetectionResult:
    def __init__(self, compatible: bool, reason: str) -> None:
        self.compatible = compatible
        self.reason = reason


def detect_fn(context: gcp.Context) -> DetectionResult:
    """Detects if the function is a Node.js 8 legacy function."""
    if os.getenv("GOOGLE_RUNTIME") != "nodejs8":
        return DetectionResult(False, "Only compatible with nodejs8")

    if os.getenv(env.FUNCTION_TARGET):
        return DetectionResult(True, f"Compatible with target {env.FUNCTION_TARGET}")

    return DetectionResult(False, f"{env.FUNCTION_TARGET} env var not set")


def build_fn(context: gcp.Context) -> None:
    """Sets up the execution environment for the function."""
    if os.getenv(env.FUNCTION_SOURCE):
        raise gcp.UserError(f"{env.FUNCTION_SOURCE} is not supported")

    # Determine function file
    fn_file = "function.js"
    index_js_exists = context.file_exists("index.js")
    if index_js_exists:
        fn_file = "index.js"

    package_json_path = context.application_root / "package.json"
    if package_json_path.exists():
        pjs = read_package_json(package_json_path)
        if pjs.get("main"):
            fn_file = pjs["main"]

    if not context.file_exists(fn_file):
        raise gcp.UserError(f"{fn_file} does not exist")

    # Syntax check
    try:
        subprocess.run(["node", "--check", fn_file], check=True, text=True)
    except subprocess.CalledProcessError as e:
        raise gcp.UserError(f"Syntax check failed: {e}") from None

    # Create layer
    layer = context.layer("legacy-worker")
    
    install_legacy_worker(context, layer)

    # Environment setup
    node_modules_path = context.application_root / "node_modules"
    if node_modules_path.exists():
        layer.launch_env["NODE_PATH"] = str(node_modules_path)
    
    target = os.getenv(env.FUNCTION_TARGET)
    if not target:
        raise gcp.InternalError(f"Required env var {env.FUNCTION_TARGET} not found")

    layer.launch_env.update({
        "X_GOOGLE_FUNCTION_NAME": target,
        "X_GOOGLE_ENTRY_POINT": target,
        "X_GOOGLE_FUNCTION_TRIGGER_TYPE": os.getenv(env.FUNCTION_SIGNATURE_TYPE, "HTTP_TRIGGER"),
        "X_GOOGLE_CODE_LOCATION": str(context.application_root),
        "X_GOOGLE_NEW_FUNCTION_SIGNATURE": "true",
        "X_GOOGLE_WORKER_PORT": 8091,
        "WORKER_PORT": 8091
    })

    # Add web process
    worker_path = layer.path / "worker.js"
    context.add_web_process(["node", str(worker_path)])


def read_package_json(package_json_path: Path) -> dict:
    """Reads and parses package.json."""
    with open(package_json_path, 'r') as f:
        return json.load(f)


def install_legacy_worker(context: gcp.Context, layer: gcp.Layer) -> None:
    """Installs the legacy worker dependencies."""
    converter_dir = context.buildpack_root / "converter"
    worker_js = converter_dir / "worker.js"
    package_json = converter_dir / "package.json"

    # Check cache
    if not (context.cache_root / "node_modules").exists():
        install_dependencies(context, layer)

    # Copy files
    shutil.copy2(worker_js, layer.path)
    shutil.copy2(package_json, layer.path)

    # Install dependencies
    subprocess.run(["npm", "install", "--quiet", "--production"], 
                  cwd=layer.path,
                  check=True)


def install_dependencies(context: gcp.Context, layer: gcp.Layer) -> None:
    """Installs worker dependencies."""
    context.log("Installing worker dependencies")
    try:
        subprocess.run(["npm", "install", "--quiet", "--production"],
                      cwd=layer.path,
                      check=True)
    except subprocess.CalledProcessError as e:
        raise gcp.UserError(f"Failed to install dependencies: {e}") from None
