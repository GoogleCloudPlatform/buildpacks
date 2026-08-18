// Copyright 2026 Google LLC
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

// Package static contains library code for the static runtimes buildpack.
package static

import (
	"os"
	"text/template"
)

// TODO(b/514251263): Parameterize the nginx config to allow for custom headers. eg: PORT etc.
const (
	// NginxConfFile is the default configuration file name for nginx in static runtimes.
	NginxConfFile = "nginx.conf"
	nginxConfTmpl = `
pid /tmp/nginx.pid;
error_log /dev/stderr notice;

events {
    worker_connections 1024;
}

http {
    include {{.MimeTypesPath}};
    default_type application/octet-stream;

    access_log /dev/stdout;

    client_body_temp_path /tmp/nginx_client_body;
    proxy_temp_path /tmp/nginx_proxy;
    fastcgi_temp_path /tmp/nginx_fastcgi;
    uwsgi_temp_path /tmp/nginx_uwsgi;
    scgi_temp_path /tmp/nginx_scgi;

    # Performance & Kernel Optimizations
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    open_file_cache max=10000 inactive=30s;
    open_file_cache_valid 60s;
    open_file_cache_min_uses 2;
    open_file_cache_errors on;

    # Compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_min_length 256;
    gzip_types
        text/plain
        text/css
        text/javascript
        application/javascript
        application/json
        application/xml
        application/wasm
        image/svg+xml
        font/ttf
        font/otf;

    # Security
    server_tokens off;

    server {
        listen 8080 default_server;
        server_name _;

        root {{.RootPath}};
        index index.html;

        # Reverse-proxy & redirect safety for Cloud Run
        absolute_redirect off;
        port_in_redirect off;

        # Block hidden dotfiles (.git, .env, .DS_Store)
        location ~ /\. {
            access_log off;
            log_not_found off;
            return 404;
        }

        {{range .Redirects}}
        location ~ {{.Pattern}} {
            return {{.Code}} {{.Target}};
        }
        {{end}}

				{{range .Rewrites}}
        location ~ {{.Pattern}} {
            rewrite {{.Pattern}} {{.Target}} break;
        }
        {{end}}

        {{range .HeaderBlocks}}
        location {{.Location}} {
            {{range .Headers}}
            add_header "{{.Name}}" "{{.Value}}";
            {{end}}
        }
        {{end}}

        # Default Fallback
        location / {
            add_header X-Content-Type-Options "nosniff" always;
            try_files $uri $uri/ $uri.html /index.html =404;
        }
    }
}
`
	// DefaultStaticNginxVersion is the default Nginx version constraint for runtimes not specified in the map.
	DefaultStaticNginxVersion = "1.24.x"
)

// NginxConfigParams holds the generic configuration parameters for templating nginx.conf.
type NginxConfigParams struct {
	RootPath      string
	MimeTypesPath string
	Rewrites      []NginxRewrite
	Redirects     []NginxRedirect
	HeaderBlocks  []NginxHeaderBlock
}

// NginxRewrite represents a single internal rewrite rule.
type NginxRewrite struct {
	Pattern string // Regex pattern (e.g., "^/api/(.*)$")
	Target  string // Destination (e.g., "http://backend/$1")
}

// NginxRedirect represents an HTTP redirect.
type NginxRedirect struct {
	Pattern string // Regex pattern
	Target  string // Destination URL
	Code    int    // HTTP Status Code (e.g. 301, 302)
}

// NginxHeader represents a single key-value HTTP header.
type NginxHeader struct {
	Name  string
	Value string
}

// NginxHeaderBlock represents a location block containing HTTP headers.
type NginxHeaderBlock struct {
	Location string        // Path matching string (e.g., "~* \.(css|js)$")
	Headers  []NginxHeader // Slice of custom header key-value pairs (ordered)
}

// WriteNginxConfig compiles the configuration template with parameters and writes it to disk.
func WriteNginxConfig(dstPath string, params NginxConfigParams) error {
	tmpl, err := template.New(NginxConfFile).Parse(nginxConfTmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, params)
}

const (
	// RuntimeStatic24 is the runtime name for static24 base image.
	RuntimeStatic24 = "static24"
)

var (
	// NginxVersionPerRuntime maps a runtime name to its specific Nginx version constraint.
	NginxVersionPerRuntime = map[string]string{
		RuntimeStatic24: "1.24.x",
	}
)

// NginxVersionConstraint returns the Nginx version constraint for the specified runtime name.
// If the runtime name is not found in the map, it returns the default Nginx version constraint.
func NginxVersionConstraint(runtimeName string) string {
	if ver, ok := NginxVersionPerRuntime[runtimeName]; ok {
		return ver
	}
	return DefaultStaticNginxVersion
}
