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
Implements php/supervisor buildpack.
The supervisor buildpack installs the config needed for PHP runtime with supervisor.
"""

import os
import os.path
import sys
from pathlib import Path
import textwrap
from jinja2 import Template

import appyaml
import flex
import gcpbuildpack as gcp
import nginx
import webconfig

# Constants
DEFAULT_FRONT_CONTROLLER = "index.php"
DEFAULT_NGINX_BINARY = "nginx"
DEFAULT_NGINX_PORT = 8080
DEFAULT_ROOT = "/workspace"
DEFAULT_NGINX_CONF_INCLUDE = "nginx-app.conf"
DEFAULT_NGINX_CONF_HTTP_INCLUDE = "nginx-http.conf"
DEFAULT_NGINX_CONF = "nginx.conf"
NGINX_LOG = "nginx.log"
DEFAULT_ADDRESS = "127.0.0.1:9000"

DEFAULT_PHP_FPM_CONF_OVERRIDE = "php-fpm.conf"
DEFAULT_DYNAMIC_WORKERS = False
DEFAULT_FPM_BINARY = "php-fpm"
DEFAULT_FPM_WORKERS = 2
PHP_FPM_PID = "php-fpm.pid"

DEFAULT_PHP_INI = "php.ini"

def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """
    Detects if supervisor package is needed.
    
    Args:
        ctx: The build context
        
    Returns:
        A tuple containing the detection result and any error
    """
    try:
        if flex.needs_supervisor_package(ctx):
            return (gcp.OptIn("supervisor package is required"), None)
        
        return (gcp.OptOut("supervisor package is not required"), None)
    except Exception as e:
        return (None, e)

def build_fn(ctx: gcp.Context) -> Exception:
    """
    Builds the supervisor configuration.
    
    Args:
        ctx: The build context
        
    Returns:
        Any error encountered during building
    """
    try:
        layer = ctx.Layer(
            "supervisor",
            gcp.CacheLayer,
            gcp.LaunchLayerUnlessSkipRuntimeLaunch,
            gcp.BuildLayer
        )
        
        if layer is None:
            raise ValueError("Failed to create layer")
            
        layer.launch_environment.default("APP_DIR", DEFAULT_ROOT)
        
        runtime_config = appyaml.php_configuration(ctx.application_root())
        
        if flex.install_supervisor(ctx, layer):
            return None
            
        overrides = webconfig.overridden_properties(ctx, runtime_config)
        webconfig.set_env_variables(layer, overrides)
        
        fpm_conf_file, error = write_fpm_config(layer.path, overrides)
        if error:
            return error
            
        supervisor_files, error = flex.supervisor_conf_files(
            ctx,
            runtime_config,
            ctx.application_root()
        )
        if error:
            return error
            
        nginx_path = os.path.join(layer.path, DEFAULT_NGINX_CONF)
        
        if not overrides.nginx_conf_override:
            nginx_server_conf, error = write_nginx_server_config(layer.path, overrides)
            if error:
                return error
                
            supervisor_nginx_conf = supervisor_nginx_config(nginx_server_conf, overrides)
            
            _, error = write_template_config_to_path(
                nginx_path,
                flex.NginxConfTemplate,
                supervisor_nginx_conf
            )
            if error:
                return error
        else:
            nginx_path = overrides.nginx_conf_override_file_name
            
        supervisor_path, error = supervisor_location(
            supervisor_files,
            nginx_path,
            fpm_conf_file.name(),
            layer.path
        )
        if error:
            return error
            
        cmd = ["supervisord", "-c", supervisor_path]
        
        ctx.add_process(gcp.WebProcess, cmd, gcp.AsDefaultProcess())
        
        return None
        
    except Exception as e:
        return e

def supervisor_location(
    supervisor_files: flex.SupervisorFiles,
    nginx_path: str,
    fpm_conf_file: str,
    layer: str
) -> tuple[str, Exception]:
    """
    Determines the location of the supervisor configuration.
    
    Args:
        supervisor_files: Supervisor file information
        nginx_path: Path to Nginx configuration
        fpm_conf_file: PHP-FPM configuration file
        layer: Layer path
        
    Returns:
        A tuple containing the supervisor path and any error
    """
    try:
        if supervisor_files.supervisor_conf_exists:
            return (supervisor_files.supervisor_conf, None)
            
        supervisor_conf = supervisor_config(fpm_conf_file, nginx_path, supervisor_files)
        supervisor_file, error = write_template_config_to_path(
            os.path.join(layer, "supervisord.conf"),
            flex.SupervisorTemplate,
            supervisor_conf
        )
        
        if error:
            return ("", error)
            
        return (supervisor_file.name(), None)
    except Exception as e:
        return ("", e)

def supervisor_nginx_config(
    nginx_server_path: str,
    overrides: webconfig.OverrideProperties
) -> flex.NginxConfig:
    """
    Creates Nginx configuration for supervisor.
    
    Args:
        nginx_server_path: Path to Nginx server configuration
        overrides: Override properties
        
    Returns:
        The Nginx configuration
    """
    conf = flex.NginxConfig(
        mime_types_path=os.path.join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
        nginx_server_conf_path=nginx_server_path
    )
    
    if overrides.nginx_http_include:
        conf.nginx_conf_http_include = overrides.nginx_http_include_file_name
        
    return conf

def write_template_config_to_path(
    path: str,
    template: Template,
    config: dict
) -> tuple[os.PathLike, Exception]:
    """
    Writes a configuration file from a template.
    
    Args:
        path: Output file path
        template: Template to use
        config: Configuration data
        
    Returns:
        A tuple containing the output file and any error
    """
    try:
        with open(path, "w") as f:
            f.write(template.render(config))
            
        return (Path(path), None)
    except Exception as e:
        return (None, e)

def write_nginx_server_config(
    path: str,
    overrides: webconfig.OverrideProperties
) -> tuple[str, Exception]:
    """
    Writes the Nginx server configuration.
    
    Args:
        path: Output directory
        overrides: Override properties
        
    Returns:
        A tuple containing the configuration file name and any error
    """
    try:
        nginx_conf = nginx_config(path, overrides)
        nginx_conf_file, error = nginx.write_nginx_config_to_path(path, nginx_conf)
        
        if error:
            return ("", error)
            
        return (nginx_conf_file.name(), None)
    except Exception as e:
        return ("", e)

def nginx_config(layer: str, overrides: webconfig.OverrideProperties) -> nginx.Config:
    """
    Creates Nginx configuration.
    
    Args:
        layer: Layer path
        overrides: Override properties
        
    Returns:
        The Nginx configuration
    """
    front_controller = DEFAULT_FRONT_CONTROLLER
    if overrides.front_controller:
        front_controller = overrides.front_controller
        
    return nginx.Config(
        port=DEFAULT_NGINX_PORT,
        front_controller_script=front_controller,
        root=os.path.join(DEFAULT_ROOT, overrides.document_root),
        app_listen_address=DEFAULT_ADDRESS
    )

def supervisor_config(
    fpm_path: str,
    nginx_path: str,
    supervisor_files: flex.SupervisorFiles
) -> flex.SupervisorConfig:
    """
    Creates supervisor configuration.
    
    Args:
        fpm_path: PHP-FPM configuration path
        nginx_path: Nginx configuration path
        supervisor_files: Supervisor file information
        
    Returns:
        The supervisor configuration
    """
    conf = flex.SupervisorConfig(
        php_fpm_conf_path=fpm_path,
        nginx_conf_path=nginx_path
    )
    
    if supervisor_files.add_supervisor_conf_exists:
        conf.supervisor_include_conf_path = os.path.join(
            DEFAULT_ROOT,
            supervisor_files.add_supervisor_conf
        )
        
    return conf

def write_fpm_config(
    path: str,
    overrides: webconfig.OverrideProperties
) -> tuple[os.PathLike, Exception]:
    """
    Writes PHP-FPM configuration.
    
    Args:
        path: Output directory
        overrides: Override properties
        
    Returns:
        A tuple containing the FPM configuration file and any error
    """
    try:
        conf = fpm_config(path, overrides)
        return nginx.write_fpm_config_to_path(path, conf)
    except Exception as e:
        return (None, e)

def fpm_config(
    layer: str,
    overrides: webconfig.OverrideProperties
) -> tuple[nginx.FPMConfig, Exception]:
    """
    Creates PHP-FPM configuration.
    
    Args:
        layer: Layer path
        overrides: Override properties
        
    Returns:
        A tuple containing the FPM configuration and any error
    """
    try:
        user = os.getlogin()
        
        conf = nginx.FPMConfig(
            pid_path=os.path.join(layer, PHP_FPM_PID),
            num_workers=DEFAULT_FPM_WORKERS,
            listen_address=DEFAULT_ADDRESS,
            dynamic_workers=DEFAULT_DYNAMIC_WORKERS,
            username=user,
            add_no_decorate_workers=True,
            use_log_limit=True
        )
        
        if overrides.php_fpm_override:
            conf.conf_override = overrides.php_fpm_override_file_name
            
        return (conf, None)
    except Exception as e:
        return (None, e)
