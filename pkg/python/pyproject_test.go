// Copyright 2024 Google LLC
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

package python

import (
	"os"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestIsPoetryProject(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    bool
		wantMsg string
	}{
		{
			name: "poetry.lock_exists",
			files: map[string]string{
				"poetry.lock": "",
			},
			want:    true,
			wantMsg: "found poetry.lock",
		},
		{
			name: "pyproject.toml_with_tool.poetry_section",
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]
name = "my-test-project"`,
			},
			want:    true,
			wantMsg: "found [tool.poetry] in pyproject.toml",
		},
		{
			name: "pyproject.toml_without_tool.poetry_section",
			files: map[string]string{
				"pyproject.toml": `[tool.other]
name = "my-test-project"`,
			},
			want:    false,
			wantMsg: "neither poetry.lock nor [tool.poetry] found",
		},
		{
			name:    "no_relevant_files_exist",
			files:   map[string]string{},
			want:    false,
			wantMsg: "pyproject.toml not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current working directory: %v", err)
			}
			if err := os.Chdir(appDir); err != nil {
				t.Fatalf("failed to change directory to %s: %v", appDir, err)
			}

			defer os.Chdir(cwd)

			for path, content := range tc.files {
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("writing file %q: %v", path, err)
				}
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			isPoetry, msg, err := IsPoetryProject(ctx)

			if err != nil {
				t.Fatalf("IsPoetryProject() got an unexpected error: %v", err)
			}
			if isPoetry != tc.want {
				t.Errorf("IsPoetryProject() = %v, want %v", isPoetry, tc.want)
			}
			if msg != tc.wantMsg {
				t.Errorf("IsPoetryProject() message = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}

func TestRequestedPoetryVersion(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "valid_requires-poetry_constraint",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry]
					requires-poetry = ">=2.1.0"
				`,
			},
			want:    ">=2.1.0",
			wantErr: false,
		},
		{
			name: "no_requires-poetry_constraint",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry]
				`,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "malformed_pyproject.toml",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry
					requires-poetry = "<2.0.0"
				`,
			},
			want:    "",
			wantErr: false,
		},
		{
			name:  "file_does_not_exist",
			files: map[string]string{
				// No pyproject.toml file
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current working directory: %v", err)
			}
			if err := os.Chdir(appDir); err != nil {
				t.Fatalf("failed to change directory to %s: %v", appDir, err)
			}
			defer os.Chdir(cwd)

			for path, content := range tc.files {
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("writing file %q: %v", path, err)
				}
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			version, err := RequestedPoetryVersion(ctx)

			if (err != nil) != tc.wantErr {
				t.Errorf("RequestedPoetryVersion() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err == nil && version != tc.want {
				t.Errorf("RequestedPoetryVersion() = %q, want %q", version, tc.want)
			}
		})
	}
}
