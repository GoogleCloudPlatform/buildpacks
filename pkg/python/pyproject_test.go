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
			appDir := setupTest(t, tc.files)

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
			wantErr: true,
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
			appDir := setupTest(t, tc.files)

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

func TestIsUVProject(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    bool
		wantMsg string
	}{
		{
			name: "uv_project_with_uv.lock",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-uv-project"`,
				"uv.lock": "",
			},
			want:    true,
			wantMsg: "found pyproject.toml and uv.lock",
		},
		{
			name: "uv_project_without_uv.lock",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-uv-project"`,
			},
			want:    true,
			wantMsg: "found pyproject.toml",
		},
		{
			// That's why Poetry order group should be before UV order group.
			name: "poetry_project",
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]
name = "my-poetry-project"`,
			},
			want:    true,
			wantMsg: "found pyproject.toml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			isUV, msg, err := IsUVProject(ctx)

			if err != nil {
				t.Fatalf("IsUVProject() got an unexpected error: %v", err)
			}
			if isUV != tc.want {
				t.Errorf("IsUVProject() = %v, want %v", isUV, tc.want)
			}
			if msg != tc.wantMsg {
				t.Errorf("IsUVProject() message = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}

func TestRequestedUVVersion(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "valid_required-version_constraint",
			files: map[string]string{
				"pyproject.toml": `
					[tool.uv]
					required-version = ">=0.1.0"
				`,
			},
			want:    ">=0.1.0",
			wantErr: false,
		},
		{
			name: "no_required-version_constraint",
			files: map[string]string{
				"pyproject.toml": `
					[tool.uv]
				`,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "malformed_pyproject.toml",
			files: map[string]string{
				"pyproject.toml": `
					[tool.uv
					required-version = "<1.0.0"
				`,
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			version, err := RequestedUVVersion(ctx)

			if (err != nil) != tc.wantErr {
				t.Errorf("RequestedUVVersion() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err == nil && version != tc.want {
				t.Errorf("RequestedUVVersion() = %q, want %q", version, tc.want)
			}
		})
	}
}

func setupTest(t *testing.T, files map[string]string) string {
	t.Helper()
	appDir := t.TempDir()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(appDir); err != nil {
		t.Fatalf("failed to change directory to %s: %v", appDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("failed to change directory back to %s: %v", cwd, err)
		}
	})

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("writing file %q: %v", path, err)
		}
	}
	return appDir
}
