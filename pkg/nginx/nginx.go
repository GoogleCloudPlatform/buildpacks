import os
from dataclasses import dataclass
from jinja2 import Template

# Package nginx contains nginx buildpack library code.

@dataclass
class FPMConfig:
    pid_path: str
    listen_address: str
    dynamic_workers: bool
    num_workers: int
    username: str
    add_no_decorate_workers: bool
    conf_override: str
    use_log_limit: bool

@dataclass
class Config:
    port: int
    root: str
    app_listen_address: str
    front_controller_script: str
    nginx_conf_include: str
    serves_static_files: bool

# Templates
phpfpm_template_str = '''
{% if UseLogLimit %}
log_limit = 262144 ; 256 kb
{% endif %}

; Send errors to stderr.
error_log = /proc/self/fd/2

log_level = warning

pid = {{ PidPath }}

; Pool configuration
[app]

; Unix user/group of processes
user = {{ Username }}
group = {{ Username }}

; The address on which to accept FastCGI requests
listen = {{ ListenAddress }}

{% if DynamicWorkers %}
; Create child processes with a dynamic policy.
pm = dynamic

; The number of child processes to be created
pm.start_servers = 1
pm.min_spare_servers = 1
pm.max_spare_servers = {{ NumWorkers }}
pm.max_children = {{ NumWorkers }}
{% else %}
; Create child processes with a static policy.
pm = static

; The number of child processes to be created
pm.max_children = {{ NumWorkers }}
{% endif %}

; Keep the environment variables of the parent.
clear_env = no

catch_workers_output = yes
{% if AddNoDecorateWorkers %}
decorate_workers_output = no
{% endif %}

{% if ConfOverride %}
include = {{ ConfOverride }}
{% endif %}
'''

phpfpm_template = Template(phpfpm_template_str)

nginx_template_str = '''
fastcgi_read_timeout 24h;

# proxy_* are not set for PHP because fastcgi is used.

upstream fast_cgi_app {
    server         {{ AppListenAddress }} fail_timeout=0;
}

server {
    listen	{{ Port }} default_server;
    listen	[::]:{{ Port }} default_server;
    server_name	"";
    root	{{ Root }};

    {% if ServesStaticFiles %}
    location / {
        try_files $uri /{{ FrontControllerScript }}$uri$is_args$args;
    }
    {% else %}
    rewrite	^/(.*)$	/{{ FrontControllerScript }}$uri;
    {% endif %}

    location	~	^/{{ FrontControllerScript }}	{
        error_log stderr;

        fastcgi_pass	fast_cgi_app;
        fastcgi_buffering	off;
        fastcgi_request_buffering	off;
        fastcgi_cache	off;
        fastcgi_store	off;
        fastcgi_intercept_errors	off;

        fastcgi_index	index.php;
        fastcgi_split_path_info	^(.+\.php)(.*)$;

        fastcgi_param	QUERY_STRING	$query_string;
        fastcgi_param	REQUEST_METHOD	$request_method;
        fastcgi_param	CONTENT_TYPE	$content_type;
        fastcgi_param	CONTENT_LENGTH	$content_length;

        fastcgi_param	SCRIPT_NAME	$fastcgi_script_name;
        fastcgi_param	SCRIPT_FILENAME	$document_root/{{ FrontControllerScript }};
        fastcgi_param	PATH_INFO	$fastcgi_path_info;
        fastcgi_param	REQUEST_URI	$request_uri;
        fastcgi_param	DOCUMENT_URI	$fastcgi_script_name;
        fastcgi_param	DOCUMENT_ROOT	$document_root;
        fastcgi_param	SERVER_PROTOCOL	$server_protocol;
        fastcgi_param	REQUEST_SCHEME	$scheme;
        if ($http_x_forwarded_proto = 'https') {
            set $https_setting 'on';
        }
        fastcgi_param	HTTPS	$https_setting if_not_empty;

        fastcgi_param	GATEWAY_INTERFACE	CGI/1.1;
        fastcgi_param	REMOTE_ADDR	$remote_addr;
        fastcgi_param	REMOTE_PORT	$remote_port;
        fastcgi_param	REMOTE_HOST	$remote_addr;
        fastcgi_param	REMOTE_USER	$remote_user;
        fastcgi_param	SERVER_ADDR	$server_addr;
        fastcgi_param	SERVER_PORT	$server_port;
        fastcgi_param	SERVER_NAME	$server_name;
        fastcgi_param X_FORWARDED_FOR $proxy_add_x_forwarded_for;
        fastcgi_param X_FORWARDED_HOST $http_x_forwarded_host;
        fastcgi_param X_FORWARDED_PROTO $http_x_forwarded_proto;
        fastcgi_param FORWARDED $http_forwarded;
    }

    {% if NginxConfInclude %}
    include {{ NginxConfInclude }};
    {% endif %}
}
'''

nginx_template = Template(nginx_template_str)

nginx_server_conf = "nginxserver.conf"
php_fpm_conf = "php-fpm.conf"

def write_nginx_config_to_path(path: str, config: Config) -> None:
    file_path = os.path.join(path, nginx_server_conf)
    parent_dir = os.path.dirname(file_path)
    if not os.path.exists(parent_dir):
        os.makedirs(parent_dir, exist_ok=True)

    with open(file_path, 'w') as f:
        rendered_content = nginx_template.render(**config.__dict__)
        f.write(rendered_content)

def write_fpm_config_to_path(path: str, config: FPMConfig) -> None:
    file_path = os.path.join(path, php_fpm_conf)
    parent_dir = os.path.dirname(file_path)
    if not os.path.exists(parent_dir):
        os.makedirs(parent_dir, exist_ok=True)

    with open(file_path, 'w') as f:
        rendered_content = phpfpm_template.render(**config.__dict__)
        f.write(rendered_content)
