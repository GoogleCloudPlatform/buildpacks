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
    access_log /dev/stdout;

    client_body_temp_path /tmp/nginx_client_body;
    proxy_temp_path /tmp/nginx_proxy;
    fastcgi_temp_path /tmp/nginx_fastcgi;
    uwsgi_temp_path /tmp/nginx_uwsgi;
    scgi_temp_path /tmp/nginx_scgi;

    server {
        listen 8080;
        root {{.RootPath}};
        index index.html;

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
            try_files $uri $uri/ /index.html;
        }
        {{end}}

        # Default Fallback
        location / {
            try_files $uri $uri/ /index.html;
        }

        absolute_redirect off;
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
