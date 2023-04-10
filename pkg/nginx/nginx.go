// Copyright 2022 Google LLC
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

// Package nginx contains nginx buildpack library code.
package nginx

import (
	"text/template"
)

// PHPFpmTemplate is a template that produces a snippet of php-fpm config that sets up the PHP with Nginx.
var PHPFpmTemplate = template.Must(template.New("phpfpm").Parse(`
; Send errors to stderr.
error_log = /proc/self/fd/2

log_level = warning

pid = {{.PidPath}}

; Pool configuration
[app]

; Unix user/group of processes
user = {{.Username}}
group = {{.Username}}

; The address on which to accept FastCGI requests
listen = {{.ListenAddress}}

{{if .DynamicWorkers}}
; Create child processes with a dynamic policy.
pm = dynamic

; The number of child processes to be created
pm.start_servers = 1
pm.min_spare_servers = 1
pm.max_spare_servers = {{.NumWorkers}}
pm.max_children = {{.NumWorkers}}
{{else}}
; Create child processes with a static policy.
pm = static

; The number of child processes to be created
pm.max_children = {{.NumWorkers}}
{{end}}

; Keep the environment variables of the parent.
clear_env = no

catch_workers_output = yes
{{if .AddNoDecorateWorkers}}
decorate_workers_output = no
{{end}}
`))

// NginxTemplate is a template that produces a snippet of nginx config that sets up the
// upstream and server for PHP. It is included in the http{} section of the config by
// the pid1 program.
var NginxTemplate = template.Must(template.New("nginx").Parse(`
fastcgi_read_timeout 24h;

# proxy_* are not set for PHP because fastcgi is used.

upstream fast_cgi_app {
	server         unix:{{.AppListenAddress}} fail_timeout=0;
}

server {
	listen	{{.Port}} default_server;
	listen	[::]:{{.Port}} default_server;
	server_name	"";
	root	{{.Root}};

	rewrite	^/(.*)$	/{{.FrontControllerScript}}$uri;

	location	~	^/{{.FrontControllerScript}}	{
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
		fastcgi_param	SCRIPT_FILENAME	$document_root/{{.FrontControllerScript}};
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
}
`))

// FPMConfig represents the content values of a php-fpm config file.
type FPMConfig struct {
	PidPath              string
	ListenAddress        string
	DynamicWorkers       bool
	NumWorkers           int
	Username             string
	AddNoDecorateWorkers bool
}

// Config represents the content values of a nginx config file.
type Config struct {
	Port                  int
	Root                  string
	AppListenAddress      string
	FrontControllerScript string
}
