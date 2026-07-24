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

"""Implements php/webconfig buildpack."""

import os
import sys
from pathlib import Path

import semver
from google.cloud.buildpacks.gcpbuildpack import gcp  # type: ignore

const = {
    "appSocket": "app.sock",
    "pid1Log": "pid1.log",
    "defaultFlexAddress": "127.0.0.1:9000",
    "defaultFrontController": "index.php",
    "defaultNginxBinary": "nginx",
    "defaultNginxPort": 8080,
    "defaultRoot": "/workspace",
    "nginxConf": "nginx.conf",
    "nginxLog": "nginx.log",
    "phpFpmPid": "php-fpm.pid",
}

overrides = {
    "nginx_conf_override": False,
    "nginx_conf_override_filename": "",
    "nginx_serves_static_files": False,
    "phpfpm_override": False,
    "phpfpm_override_filename": "",
    "front_controller": "",
    "document_root": "",
    "nginx_server_conf_include": False,
    "nginx_server_conf_include_filename": "",
    "nginx_http_include": False,
    "nginx_http_include_filename": "",
}

def DetectFn():
    """Detect function for the buildpack."""
    return gcp.OptInAlways()

def BuildFn(context):
    """Build function for the buildpack."""
    # Create webconfig layer
    layer = context.Layer("webconfig", gcp.LaunchLayer)
    
    if os.getenv("FLEX_ENV"):
        runtime_config = appyaml.PhpConfiguration(context.ApplicationRoot())
        overrides.update(webconfig.OverriddenProperties(context, runtime_config))
        webconfig.SetEnvVariables(layer, overrides)

    # Handle custom nginx config
    custom_nginx_conf = os.getenv(php.CustomNginxConfig)
    if custom_nginx_conf:
        overrides["nginx_conf_override"] = True
        overrides["nginx_conf_override_filename"] = Path(defaultRoot) / custom_nginx_conf

    # Set nginx serves static files
    overrides["nginx_serves_static_files"] = env.IsPresentAndTrue(php.NginxServesStaticFiles)

    # Write FPM config
    fpm_config_file = writeFpmConfig(context, layer.Path, overrides)
    
    # Write nginx server config
    nginx_server_config_file = writeNginxServerConfig(layer.Path, overrides)

    # Handle Procfile and entrypoint
    proc_exists = context.FileExists("Procfile")
    entrypoint_exists = os.getenv(env.Entrypoint) is not None

    if not proc_exists and not entrypoint_exists:
        capability = context.Capability(php.WebConfigCapability)
        if capability:
            configurer = capability.Get()
            return configurer.Configure(context)
        
        # Set up command
        cmd = [
            os.path.join(os.getenv("PID1_DIR"), "pid1"),
            "--nginxBinaryPath", defaultNginxBinary,
            "--nginxErrLogFilePath", os.path.join(layer.Path, nginxLog),
            "--customAppCmd", f"{defaultFPMBinary} -R --nodaemonize --fpm-config {fpm_config_file.name}",
            "--pid1LogFilePath", os.path.join(layer.Path, pid1Log),
            "--mimeTypesPath", os.path.join("/layers/google.utils.nginx/nginx", "conf/mime.types")
        ]
        
        add_args = addNginxConfCmdArgs(layer.Path, nginx_server_config_file.name, overrides)
        cmd.extend(add_args)

        context.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

def getInstalledPhpVersion(context):
    """Get installed PHP version."""
    version = php.ExtractVersion(context)
    resolved_version = runtime.ResolveVersion(
        context,
        php.GetInstallableRuntime(context),
        version,
        runtime.OSForStack(context)
    )
    return resolved_version

def isVersion73OrGreater(context):
    """Check if PHP version is 7.3 or greater."""
    version = getInstalledPhpVersion(context)
    if runtime.IsReleaseCandidate(version):
        return True, None
    constraint = semver.Constraint(">=7.3.0")
    sv = semver.Version.parse(version)
    return constraint.matches(sv), None

def writeFpmConfig(context, path, overrides):
    """Write FPM configuration."""
    is_73_plus, _ = isVersion73OrGreater(context)
    config = fpmConfig(path, is_73_plus, overrides)
    return nginx.WriteFpmConfigToPath(path, config)

def fpmConfig(layer, is_73_plus, overrides):
    """Generate FPM configuration."""
    user = os.getenv("USER")
    
    config = {
        "pid_path": os.path.join(layer, phpFpmPid),
        "num_workers": defaultFPMWorkers,
        "listen_address": os.path.join(layer, appSocket),
        "dynamic_workers": defaultDynamicWorkers,
        "username": user,
        "add_no_decorate_workers": is_73_plus,
        "use_log_limit": is_73_plus
    }

    if os.getenv("FLEX_ENV"):
        config["listen_address"] = defaultFlexAddress

    if overrides.get("phpfpm_override"):
        config["conf_override"] = overrides.get("phpfpm_override_filename")

    return config

def addNginxConfCmdArgs(path, nginx_server_conf_file_name, overrides):
    """Add Nginx configuration command arguments."""
    args = []
    if os.getenv("FLEX_ENV"):
        args.append("--customAppPort")
        args.append("9000")
    else:
        args.append("--customAppSocket")
        args.append(os.path.join(path, appSocket))

    if overrides.get("nginx_conf_override"):
        args.extend(["--nginxConfigPath", overrides["nginx_conf_override_filename"]])
        return args

    args.extend([
        "--nginxConfigPath",
        os.path.join(path, nginxConf),
        "--serverConfigPath",
        nginx_server_conf_file_name
    ])

    if overrides.get("nginx_http_include"):
        args.extend([
            "--httpIncludeConfigPath",
            overrides["nginx_http_include_filename"]
        ])

    return args

def nginxConfig(layer, overrides):
    """Generate Nginx configuration."""
    front_controller = defaultFrontController
    if overrides.get("front_controller"):
        front_controller = overrides["front_controller"]

    root = defaultRoot
    if overrides.get("document_root"):
        root = os.path.join(defaultRoot, overrides["document_root"])

    config = {
        "port": defaultNginxPort,
        "front_controller_script": front_controller,
        "root": root,
        "app_listen_address": f"unix:{os.path.join(layer, appSocket)}"
    }

    if os.getenv("FLEX_ENV"):
        config["app_listen_address"] = defaultFlexAddress

    if overrides.get("nginx_server_conf_include"):
        config["nginx_conf_include"] = overrides["nginx_server_conf_include_filename"]

    return config

def writeNginxServerConfig(path, overrides):
    """Write Nginx server configuration."""
    config = nginxConfig(path, overrides)
    return nginx.WriteNginxConfigToPath(path, config)
