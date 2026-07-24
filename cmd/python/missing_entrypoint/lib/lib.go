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

"""Implements python/missing-entrypoint buildpack."""

import os
import pathlib
from typing import Optional, Tuple

import buildpack.env as env
import buildpack.gcpbuildpack as gcp
import buildpack.python as python
import buildpack.runtime as runtime

gunicorn = "gunicorn"
uvicorn = "uvicorn"
gradio = "gradio"
streamlit = "streamlit"
fastapi_standard = "fastapi[standard]"
requirements = "requirements.txt"
pyproject_toml = "pyproject.toml"
google_adk = "google-adk"


def detect(ctx: gcp.Context) -> Tuple[gcp.DetectResult, Optional[str]]:
    """Detect if this buildpack should be applied."""
    override_result = runtime.check_override("python")
    if override_result is not None:
        return override_result

    has_py_files = any(path.suffix == ".py" for path in pathlib.Path(".").iterdir())
    if not has_py_files:
        return gcp.OptOut("no .py files found"), None
    return gcp.OptIn("found .py files"), None


def build(ctx: gcp.Context) -> None:
    """Build the application with the appropriate entrypoint."""
    main_exists = os.path.exists("main.py")
    app_exists = os.path.exists("app.py")

    pyproject_exists = os.path.exists(pyproject_toml)

    supports_smart_default, err = python.supports_smart_default_entrypoint(ctx)
    if err is not None:
        raise err

    adk_present = False
    if supports_smart_default:
        adk_present, err = is_adk_present(ctx)
        if err is not None:
            raise err

    if not main_exists and not app_exists and not pyproject_exists and not adk_present:
        raise gcp.UserError(
            "for Python, provide a main.py or app.py file or set an entrypoint with "
            f"{env.Entrypoint} env var or by creating a Procfile"
        )

    # Default to app:app if both exist
    py_module = "app:app" if not main_exists else "main:app"
    py_file = "app.py" if not main_exists else "main.py"

    webserver_or_framework = gunicorn
    cmd = [gunicorn, "-b", ":8080", py_module]

    if supports_smart_default:
        cmd, webserver_or_framework, err = smart_default_entrypoint(
            ctx, py_module, py_file
        )
        if err is not None:
            raise err

    if python.is_pyproject_enabled(ctx):
        script_cmd, err = python.get_script_command(ctx)
        if err is not None:
            raise err

        is_poetry, _, err = python.is_poetry_project(ctx)
        if err is not None:
            raise err

        is_uv, _, err = python.is_uv_pyproject(ctx)
        if err is not None:
            raise err

        if script_cmd is not None:
            if is_poetry:
                cmd = ["poetry", "run"] + script_cmd
            elif is_uv:
                cmd = ["uv", "run"] + script_cmd
            else:
                cmd = script_cmd
            webserver_or_framework = ""
        else:
            if not main_exists and not app_exists and not adk_present:
                raise gcp.UserError(
                    "for Python with pyproject.toml, provide a main.py or app.py file "
                    "or a script command in pyproject.toml or set an entrypoint with "
                    f"{env.Entrypoint} env var or by creating a Procfile"
                )

    cmd = apply_dev_sync(ctx, cmd, webserver_or_framework)

    # Rewrite entrypoint for specific environments (e.g. Maker)
    cmd, err = python.adapt_entrypoint(ctx, cmd, script_cmd)
    if err is not None:
        raise err

    ctx.warn(f"Setting default entrypoint: {cmd}")
    ctx.add_process(gcp.WebProcess, cmd, gcp.AsDefaultProcess())


def smart_default_entrypoint(
    ctx: gcp.Context, py_module: str, py_file: str
) -> Tuple[list[str], str, Optional[Exception]]:
    """Determine the appropriate web server based on installed packages."""
    # Priority order: gunicorn > uvicorn > gradio > streamlit
    g_present, err = python.package_present(ctx, gunicorn)
    if err is not None:
        return [], "", err
    if g_present:
        return [gunicorn, "-b", ":8080", py_module], gunicorn, None

    u_present, err = python.package_present(ctx, uvicorn)
    if err is not None:
        return [], "", err
    if u_present:
        return [
            uvicorn,
            py_module,
            "--port",
            "8080",
            "--host",
            "0.0.0.0",
        ], uvicorn, None

    fastapi_present, err = python.package_present(ctx, fastapi_standard)
    if err is not None:
        return [], "", err
    if fastapi_present:
        return [
            uvicorn,
            py_module,
            "--port",
            "8080",
            "--host",
            "0.0.0.0",
        ], uvicorn, None

    gradio_present, err = python.package_present(ctx, gradio)
    if err is not None:
        return [], "", err
    if gradio_present:
        add_gradio_env_var_layer(ctx)
        return [python.executable(), py_file], gradio, None

    streamlit_present, err = python.package_present(ctx, streamlit)
    if err is not None:
        return [], "", err
    if streamlit_present:
        return [
            streamlit,
            "run",
            py_file,
            "--server.address",
            "0.0.0.0",
            "--server.port",
            "8080",
        ], streamlit, None

    adk_present, err = is_adk_present(ctx)
    if err is not None:
        return [], "", err
    if adk_present:
        return [
            google_adk,
            "api_server",
            "--port",
            "8080",
            "--host",
            "0.0.0.0",
        ], google_adk, None

    return [gunicorn, "-b", ":8080", py_module], gunicorn, None


def add_gradio_env_var_layer(ctx: gcp.Context) -> None:
    """Add environment variables for Gradio."""
    layer, err = ctx.layer("gradio-env-var", gcp.CacheLayer | gcp.LaunchLayer)
    if err is not None:
        raise err
    layer.launch_environment["GRADIO_SERVER_NAME"] = "0.0.0.0"
    layer.launch_environment["GRADIO_SERVER_PORT"] = "8080"


def is_adk_present(ctx: gcp.Context) -> Tuple[bool, Optional[Exception]]:
    """Check if google-adk is present."""
    adk_present, err = python.package_present(ctx, google_adk)
    return adk_present, err


def apply_dev_sync(
    ctx: gcp.Context, cmd: list[str], webserver_or_framework: str
) -> list[str]:
    """Modify the entrypoint command for dev sync mode."""
    dev_sync, err = env.is_dev_sync()
    if err is not None:
        ctx.warn(f"Unable to determine dev sync status: {err}")

    if not dev_sync:
        return cmd

    if webserver_or_framework == gunicorn:
        return [gunicorn, "--reload"] + cmd[1:]

    return cmd
