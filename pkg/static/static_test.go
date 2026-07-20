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

package static

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteNginxConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, NginxConfFile)

	params := NginxConfigParams{
		RootPath:      "/my/app/root",
		MimeTypesPath: "/opt/nginx/conf/mime.types",
		HeaderBlocks: []NginxHeaderBlock{
			{
				Location: "/static",
				Headers: []NginxHeader{
					{Name: "Cache-Control", Value: "public, max-age=31536000"},
					{Name: "X-Custom", Value: "value"},
				},
			},
		},
		Redirects: []NginxRedirect{
			{
				Pattern: "/old-path",
				Target:  "/new-path",
				Code:    301,
			},
		},
		Rewrites: []NginxRewrite{
			{
				Pattern: "^/api/(.*)$",
				Target:  "/$1",
			},
		},
	}

	if err := WriteNginxConfig(dstPath, params); err != nil {
		t.Fatalf("WriteNginxConfig() error = %v", err)
	}

	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", dstPath, err)
	}

	got := string(content)
	if !strings.Contains(got, "root /my/app/root;") {
		t.Errorf("WriteNginxConfig() output = %q; missing root directive", got)
	}
	if !strings.Contains(got, "include /opt/nginx/conf/mime.types;") {
		t.Errorf("WriteNginxConfig() output = %q; missing mime.types include", got)
	}
	if !strings.Contains(got, "worker_connections 1024;") {
		t.Errorf("WriteNginxConfig() output = %q; missing worker_connections", got)
	}

	// Verify header blocks.
	if !strings.Contains(got, "location /static {") {
		t.Errorf("WriteNginxConfig() output = %q; missing /static location block for custom headers", got)
	}
	if !strings.Contains(got, `add_header "Cache-Control" "public, max-age=31536000";`) {
		t.Errorf("WriteNginxConfig() output = %q; missing Cache-Control header", got)
	}
	if !strings.Contains(got, `add_header "X-Custom" "value";`) {
		t.Errorf("WriteNginxConfig() output = %q; missing X-Custom header", got)
	}

	// Verify redirects.
	if !strings.Contains(got, "location ~ /old-path {") {
		t.Errorf("WriteNginxConfig() output = %q; missing redirect location block", got)
	}
	if !strings.Contains(got, "return 301 /new-path;") {
		t.Errorf("WriteNginxConfig() output = %q; missing return statement for redirect", got)
	}

	// Verify rewrites.
	if !strings.Contains(got, `location ~ ^/api/(.*)$ {`) {
		t.Errorf("WriteNginxConfig() output = %q; missing rewrite location block", got)
	}
	if !strings.Contains(got, `rewrite ^/api/(.*)$ /$1 break;`) {
		t.Errorf("WriteNginxConfig() output = %q; missing rewrite statement", got)
	}
}

func TestNginxVersionConstraint(t *testing.T) {
	testCases := []struct {
		name        string
		runtimeName string
		want        string
	}{
		{
			name:        "static24_runtime",
			runtimeName: RuntimeStatic24,
			want:        "1.24.x",
		},
		{
			name:        "php_runtime",
			runtimeName: "php",
			want:        DefaultStaticNginxVersion,
		},
		{
			name:        "buildpacks_runtime",
			runtimeName: "buildpacks",
			want:        DefaultStaticNginxVersion,
		},
		{
			name:        "unknown_runtime",
			runtimeName: "unknown",
			want:        DefaultStaticNginxVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NginxVersionConstraint(tc.runtimeName); got != tc.want {
				t.Errorf("NginxVersionConstraint(%q) = %q, want %q", tc.runtimeName, got, tc.want)
			}
		})
	}
}
