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

func TestGetField(t *testing.T) {
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
			if tc.path != "" {
				fp := filepath.Join(tempRoot, tc.path)
				os.WriteFile(fp, []byte(tc.content), 0664)
				for _, env := range tc.env {
					v := strings.Split(env, "=")
					t.Setenv(v[0], filepath.Join(tempRoot, v[1]))
				}
			}

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
