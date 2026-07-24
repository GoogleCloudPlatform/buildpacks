from pydantic import BaseModel, Field
import os.path
from typing import Optional, Tuple, Dict
import asyncio

class OverrideProperties(BaseModel):
    ComposerFlags: str = ""
    DocumentRoot: str = "/workspace"
    FrontController: str = "index.php"
    NginxConfOverride: bool = False
    NginxConfOverrideFileName: str = ""
    NginxServerConfInclude: bool = False
    NginxServerConfIncludeFileName: str = ""
    NginxHTTPInclude: bool = False
    NginxHTTPIncludeFileName: str = ""
    PHPFPMOverride: bool = False
    PHPFPMOverrideFileName: str = ""
    PHPIniOverride: bool = False
    PHPIniOverrideFileName: str = ""
    NginxServesStaticFiles: bool = False

DEFAULT_ROOT = "/workspace"
DEFAULT_NGINX_CONF_INCLUDE = "nginx-app.conf"
DEFAULT_NGINX_HTTP_INCLUDE = "nginx-http.conf"
DEFAULT_NGINX_CONF = "nginx.conf"
DEFAULT_PHP_FPM_CONF_OVERRIDE = "php-fpm.conf"
DEFAULT_PHP_INI = "php.ini"

async def file_exists(path: str) -> bool:
    try:
        return await asyncio.to_thread(os.path.exists, path)
    except Exception:
        return False

async def override_properties(ctx: Dict, config_value: str, default_file: str) -> Tuple[bool, str]:
    if config_value:
        return (True, os.path.join(DEFAULT_ROOT, config_value))

    exists = await file_exists(default_file)
    if exists:
        return (True, os.path.join(DEFAULT_ROOT, default_file))
    return (False, "")

async def overridden_properties(runtime_config: Dict) -> OverrideProperties:
    php_ini_result = await override_properties({}, runtime_config.get("PHPIniOverride", ""), DEFAULT_PHP_INI)
    php_fpm_result = await override_properties({}, runtime_config.get("PHPFPMConfOverride", ""), DEFAULT_PHP_FPM_CONF_OVERRIDE)
    nginx_conf_result = await override_properties({}, runtime_config.get("NginxConfOverride", ""), DEFAULT_NGINX_CONF)
    nginx_server_include_result = await override_properties({}, runtime_config.get("NginxConfInclude", ""), DEFAULT_NGINX_CONF_INCLUDE)
    nginx_http_include_result = await override_properties({}, runtime_config.get("NginxConfHTTPInclude", ""), DEFAULT_NGINX_HTTP_INCLUDE)

    return OverrideProperties(
        ComposerFlags=runtime_config.get("ComposerFlags", ""),
        DocumentRoot=runtime_config.get("DocumentRoot", "/workspace"),
        FrontController=runtime_config.get("FrontControllerFile", "index.php"),
        PHPIniOverride=php_ini_result[0],
        PHPIniOverrideFileName=php_ini_result[1],
        PHPFPMOverride=php_fpm_result[0],
        PHPFPMOverrideFileName=php_fpm_result[1],
        NginxConfOverride=nginx_conf_result[0],
        NginxConfOverrideFileName=nginx_conf_result[1],
        NginxServerConfInclude=nginx_server_include_result[0],
        NginxServerConfIncludeFileName=nginx_server_include_result[1],
        NginxHTTPInclude=nginx_http_include_result[0],
        NginxHTTPIncludeFileName=nginx_http_include_result[1]
    )

class Layer:
    def __init__(self):
        self.build_env = {}
        self.launch_env = {}

async def set_env_variables(layer: Layer, props: OverrideProperties) -> None:
    if props.ComposerFlags:
        layer.build_env["COMPOSER_ARGS"] = props.ComposerFlags

    if props.PHPIniOverride:
        layer.launch_env["PHPRC"] = props.PHPIniOverrideFileName
