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

package appyaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetEntrypointIfExists(t *testing.T) {
	testCases := []struct {
		name    string
		env     []string
		path    string
		content []byte
		want    string
		wantErr bool
	}{
		{
			name: "no env var",
			want: "",
		},
		{
			name:    "no env var, has app.yaml",
			path:    "app.yaml",
			content: []byte("entrypoint: my entrypoint"),
			want:    "",
		},
		{
			name:    "valid entrypoint",
			env:     []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			path:    "app.yaml",
			content: []byte("entrypoint: my entrypoint"),
			want:    "my entrypoint",
		},
		{
			name:    "mismatch env var and file name",
			env:     []string{"GAE_APPLICATION_YAML_PATH=foo.yaml"},
			path:    "bar.yaml",
			content: []byte("entrypoint: my entrypoint"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing entrypoint",
			env:     []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			path:    "app.yaml",
			content: []byte("foo: bar"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "multiple entries",
			env:     []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			path:    "app.yaml",
			content: []byte("foo: bar\nentrypoint: my entrypoint"),
			want:    "my entrypoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			writeFile(tc.path, tempRoot, tc.content, tc.env, t)

			got, err := EntrypointIfExists(tempRoot)

			if err != nil != tc.wantErr {
				t.Fatalf("got err=%t, want err=%t: %v", err != nil, tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("EntrypointIfExists returns %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPhpConfiguration(t *testing.T) {
	testCases := []struct {
		name    string
		env     []string
		path    string
		content []byte
		want    RuntimeConfig
		wantErr bool
	}{
		{
			name: "valid runtime_config",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			path: "app.yaml",
			content: []byte(`
runtime_config:
 document_root: web
`),
			want: RuntimeConfig{DocumentRoot: "web"},
		},
		{
			name: "missing runtime_config",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			path: "app.yaml",
			want: RuntimeConfig{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			writeFile(tc.path, tempRoot, tc.content, tc.env, t)

			got, err := PhpConfiguration(tempRoot)

			if err != nil != tc.wantErr {
				t.Fatalf("got err=%t, want err=%t: %v", err != nil, tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("PhpConfiguration returns %q, want %q", got, tc.want)
			}
		})
	}
}

func writeFile(path, root string, content []byte, envs []string, t *testing.T) {
	if path != "" {
		fp := filepath.Join(root, path)
		os.WriteFile(fp, []byte(content), 0664)
		for _, env := range envs {
			v := strings.Split(env, "=")
			t.Setenv(v[0], filepath.Join(root, v[1]))
		}
	}
}
