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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

// testNginxConfigParams exercises every section of the nginx.conf templates.
var testNginxConfigParams = NginxConfigParams{
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

// TestWriteNginxConfigGolden pins the output of every released template version. If this test
// fails for an existing version, do not update its golden file: add a new version instead. The
// generated output is written to TEST_UNDECLARED_OUTPUTS_DIR to help create the golden file for
// a new version.
func TestWriteNginxConfigGolden(t *testing.T) {
	for version := range nginxConfTemplates {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			dstPath := filepath.Join(t.TempDir(), NginxConfFile)
			if err := WriteNginxConfig(dstPath, version, testNginxConfigParams); err != nil {
				t.Fatalf("WriteNginxConfig(%d) error = %v", version, err)
			}
			got, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("os.ReadFile(%q) error = %v", dstPath, err)
			}

			goldenName := fmt.Sprintf("nginx_v%d_expected.conf", version)
			if outDir := os.Getenv("TEST_UNDECLARED_OUTPUTS_DIR"); outDir != "" {
				if err := os.WriteFile(filepath.Join(outDir, goldenName), got, 0644); err != nil {
					t.Logf("Writing generated config to undeclared outputs: %v", err)
				}
			}
			want, err := os.ReadFile(testdata.MustGetPath(filepath.Join("testdata", goldenName)))
			if err != nil {
				t.Fatalf("Reading golden file %s: %v (every version needs a golden file)", goldenName, err)
			}
			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("WriteNginxConfig(%d) output changed (-want +got):\n%s", version, diff)
			}
		})
	}
}

func TestWriteNginxConfigUnsupportedVersion(t *testing.T) {
	dstPath := filepath.Join(t.TempDir(), NginxConfFile)
	if err := WriteNginxConfig(dstPath, NginxConfigVersion(0), testNginxConfigParams); err == nil {
		t.Errorf("WriteNginxConfig(0) succeeded, want error")
	}
	if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
		t.Errorf("WriteNginxConfig(0) created %s, want no file", dstPath)
	}
}

func TestWriteNginxConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, NginxConfFile)

	if err := WriteNginxConfig(dstPath, NginxConfigV1, testNginxConfigParams); err != nil {
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

	// Verify smart defaults.
	if !strings.Contains(got, "gzip on;") {
		t.Errorf("WriteNginxConfig() output = %q; missing gzip directive", got)
	}
	if !strings.Contains(got, "gzip_static on;") {
		t.Errorf("WriteNginxConfig() output = %q; missing gzip_static directive", got)
	}
	if !strings.Contains(got, "gzip_comp_level 1;") {
		t.Errorf("WriteNginxConfig() output = %q; missing gzip_comp_level directive", got)
	}
	if !strings.Contains(got, "gzip_min_length 8192;") {
		t.Errorf("WriteNginxConfig() output = %q; missing gzip_min_length directive", got)
	}
	if !strings.Contains(got, "sendfile on;") {
		t.Errorf("WriteNginxConfig() output = %q; missing sendfile directive", got)
	}
	if !strings.Contains(got, "server_tokens off;") {
		t.Errorf("WriteNginxConfig() output = %q; missing server_tokens directive", got)
	}
	if !strings.Contains(got, "try_files $uri $uri/ $uri.html /index.html =404;") {
		t.Errorf("WriteNginxConfig() output = %q; missing clean URLs try_files directive", got)
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
