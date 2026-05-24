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
events {
    worker_connections 1024;
}

http {
    include {{.MimeTypesPath}};
    server {
        listen 8080;
        root {{.RootPath}};
        index index.html;

        location / {
            try_files $uri $uri/ /index.html;
        }
    }
}
`
	// DefaultStaticNginxVersion is the default Nginx version constraint for runtimes not specified in the map.
	DefaultStaticNginxVersion = "1.30.x"
)

// NginxConfigParams holds the runtime configuration parameters for templating nginx.conf.
type NginxConfigParams struct {
	RootPath      string
	MimeTypesPath string
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
		RuntimeStatic24: "1.30.x",
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
