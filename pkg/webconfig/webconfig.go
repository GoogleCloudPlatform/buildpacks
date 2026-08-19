# Package webconfig allows users to override some configuration properties.

import os
from dataclasses import dataclass
from typing import Optional

DEFAULT_ROOT = "/workspace"

# nginx
DEFAULT_NGINX_CONF_INCLUDE = "nginx-app.conf"
DEFAULT_NGINX_CONF_HTTP_INCLUDE = "nginx-http.conf"
DEFAULT_NGINX_CONF = "nginx.conf"

# php-fpm
DEFAULT_PHP_FPM_CONF_OVERRIDE = "php-fpm.conf"
DEFAULT_PHP_INI = "php.ini"


@dataclass
class OverrideProperties:
    # ComposerFlags overrides the composer arguments.
    composer_flags: str = ""

    # DocumentRoot specifies the DOCUMENT_ROOT for nginx and PHP.
    document_root: str = ""

    # FrontController is default PHP file name for directory access.
    front_controller: str = ""

    # NginxConfOverride boolean if user-provided nginx config exists.
    nginx_conf_override: bool = False

    # NginxConfOverrideFileName name of the user-provided nginx config.
    nginx_conf_override_file_name: str = ""

    # NginxServerConfInclude boolean if partial nginx config exists to be included in the server section.
    nginx_server_conf_include: bool = False

    # NginxServerConfIncludeFileName name of the partial nginx config to be included in the server section.
    nginx_server_conf_include_file_name: str = ""

    # NginxHTTPInclude boolean if partial nginx config exists to be included in the http section.
    nginx_http_include: bool = False

    # NginxHTTPIncludeFileName name of the partial nginx config to be included in the http section.
    nginx_http_include_file_name: str = ""

    # PHPFPMOverride boolean to check if user-provided php-fpm config exists.
    php_fpm_override: bool = False

    # PHPFPMOverrideFileName name of the user-provided php-fpm config file.
    php_fpm_override_file_name: str = ""

    # PHPIniOverride boolean to check if user-provided php ini config exists.
    php_ini_override: bool = False

    # PHPIniOverrideFileName name of the user-provided php ini config.
    php_ini_override_file_name: str = ""

    # NginxServesStaticFiles whether Nginx also serves static files for matching URIs.
    nginx_serves_static_files: bool = False


def override_properties(ctx, config_value: str, default_file: str) -> tuple[bool, str]:
    if config_value:
        return True, os.path.join(DEFAULT_ROOT, config_value)

    try:
        file_exists = ctx.file_exists(default_file)
        if file_exists:
            return True, os.path.join(DEFAULT_ROOT, default_file)
    except Exception as e:
        print(f"Error checking file existence: {e}")

    return False, ""


def overridden_properties(ctx, runtime_config) -> OverrideProperties:
    php_ini_override, php_ini.OverrideFileName = override_properties(
        ctx,
        runtime_config.php_ini_override,
        DEFAULT_PHP_INI
    )

    php_fpm_override, php_fpm_override_file_name = override_properties(
        ctx,
        runtime_config.php_fpm_conf_override,
        DEFAULT_PHP_FPM_CONF_OVERRIDE
    )

    nginx_conf_override, nginx_conf_override_file_name = override_properties(
        ctx,
        runtime_config.nginx_conf_override,
        DEFAULT_NGINX_CONF
    )

    nginx_server_conf_include, nginx_server_conf_include_file_name = override_properties(
        ctx,
        runtime_config.nginx_conf_include,
        DEFAULT_NGINX_CONF_INCLUDE
    )

    nginx_http_include, nginx_http_include_file_name = override_properties(
        ctx,
        runtime_config.nginx_conf_http_include,
        DEFAULT_NGINX_CONF_HTTP_INCLUDE
    )

    return OverrideProperties(
        composer_flags=runtime_config.composer_flags,
        document_root=runtime_config.document_root,
        front_controller=runtime_config.front_controller_file,
        php_ini_override=php_ini_override,
        php_ini_override_file_name=php_ini.OverrideFileName,
        php_fpm_override=php_fpm_override,
        php_fpm_override_file_name=php_fpm_override_file_name,
        nginx_conf_override=nginx_conf_override,
        nginx_conf_override_file_name=nginx_conf_override_file_name,
        nginx_server_conf_include=nginx_server_conf_include,
        nginx_server_conf_include_file_name=nginx_server_conf_include_file_name,
        nginx_http_include=nginx_http_include,
        nginx_http_include_file_name=nginx_http_include_file_name
    )


def set_env_variables(layer, props: OverrideProperties):
    if props.composer_flags:
        layer.build_environment["composer_args"] = props.composer_flags

    if props.php_ini_override:
        layer.launch_environment["phprc"] = props.php_ini_override_file_name
