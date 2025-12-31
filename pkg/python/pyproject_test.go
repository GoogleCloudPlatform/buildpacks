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
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
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
				"poetry.lock":    "",
				"pyproject.toml": "",
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

func TestIsUVPyproject(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		env     map[string]string
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
			wantMsg: "found pyproject.toml and GOOGLE_PYTHON_PACKAGE_MANAGER is not set, using uv as default package manager",
		},
		{
			name: "uv_project_without_uv.lock_with_uv_package_manager_env_var_set",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-uv-project"`,
			},
			env: map[string]string{
				"GOOGLE_PYTHON_PACKAGE_MANAGER": "uv",
			},
			want:    true,
			wantMsg: "found pyproject.toml, using uv because GOOGLE_PYTHON_PACKAGE_MANAGER is set to 'uv'",
		},
		{
			name: "uv_project_without_uv.lock_with_pip_package_manager_env_var_set",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-uv-project"`,
			},
			env: map[string]string{
				"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip",
			},
			want:    false,
			wantMsg: "found pyproject.toml, but GOOGLE_PYTHON_PACKAGE_MANAGER is not set to 'uv'",
		},
		{
			name: "uv_project_with_uv.lock_with_pip_package_manager_env_var_set",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-uv-project"`,
				"uv.lock": "",
			},
			env: map[string]string{
				"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip",
			},
			want:    true,
			wantMsg: "found pyproject.toml and uv.lock",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)

			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			isUV, msg, err := IsUVPyproject(ctx)

			if err != nil {
				t.Fatalf("IsUVPyproject() got an unexpected error: %v", err)
			}
			if isUV != tc.want {
				t.Errorf("IsUVPyproject() = %v, want %v", isUV, tc.want)
			}
			if msg != tc.wantMsg {
				t.Errorf("IsUVPyproject() message = %q, want %q", msg, tc.wantMsg)
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

func TestGetScriptCommand(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		want    []string
		wantErr bool
	}{
		{
			name: "poetry_single_script_found",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry.scripts]
					start_app = "my_app.main:run"
				`,
			},
			want:    []string{"start_app"},
			wantErr: false,
		},
		{
			name: "poetry_multiple_scripts_returns_start",
			files: map[string]string{
				"pyproject.toml": `
          [tool.poetry.scripts]
          dev = "my_app.dev:run"
          start = "my_app.main:run"
          lint = "my_app.lint:run"
        `,
			},
			want:    []string{"start"},
			wantErr: false,
		},
		{
			name: "poetry_multiple_scripts_no_start_returns_nil",
			files: map[string]string{
				"pyproject.toml": `
          [tool.poetry.scripts]
          dev = "my_app.dev:run"
          lint = "my_app.lint:run"
        `,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "project_single_script_found",
			files: map[string]string{
				"pyproject.toml": `
          [project.scripts]
          start_now = "my_app.main:run"
        `,
			},
			want:    []string{"start_now"},
			wantErr: false,
		},
		{
			name: "project_multiple_scripts_returns_start",
			files: map[string]string{
				"pyproject.toml": `
          [project.scripts]
          dev = "my_app.dev:run"
          start = "my_app.main:run"
        `,
			},
			want:    []string{"start"},
			wantErr: false,
		},
		{
			name: "project_multiple_scripts_no_start_returns_nil",
			files: map[string]string{
				"pyproject.toml": `
          [project.scripts]
          dev = "my_app.dev:run"
          lint = "my_app.main:run"
        `,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "poetry_single_script_takes_precedence_over_project_start",
			files: map[string]string{
				"pyproject.toml": `
          [tool.poetry.scripts]
          start1 = "my_app.poetry:run"
          [project.scripts]
          start2 = "my_app.project:run"
          start = "my_app.project:start"
        `,
			},
			want:    []string{"start1"},
			wantErr: false,
		},

		{
			name: "no_scripts_section",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					name = "my-app"
				`,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty_scripts_section",
			files: map[string]string{
				"pyproject.toml": `
					[project.scripts]
				`,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "file_does_not_exist",
			files:   map[string]string{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "malformed_pyproject.toml",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry.scripts
					start = "my_app.main:run"
				`,
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)
			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))

			cmd, err := GetScriptCommand(ctx)

			if (err != nil) != tc.wantErr {
				t.Errorf("GetScriptCommand() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(cmd, tc.want) {
				t.Errorf("GetScriptCommand() = %v, want %v", cmd, tc.want)
			}
		})
	}
}

func TestIsPipPyproject(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   map[string]string
		want  bool
	}{
		{
			name: "pip_pyproject_is_enabled_on_gcp",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			env: map[string]string{
				env.PythonPackageManager:  "pip",
				env.XGoogleTargetPlatform: "gcp",
			},
			want: true,
		},
		{
			name: "disabled_when_requirements_txt_exists",
			files: map[string]string{
				"pyproject.toml":   "[project]",
				"requirements.txt": "flask",
			},
			env: map[string]string{
				env.PythonPackageManager:  "pip",
				env.ReleaseTrack:          "BETA",
				env.XGoogleTargetPlatform: "gcp",
			},
			want: false,
		},
		{
			name: "disabled_when_package_manager_is_uv",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			env: map[string]string{
				env.PythonPackageManager:  "uv",
				env.ReleaseTrack:          "BETA",
				env.XGoogleTargetPlatform: "gcp",
			},
			want: false,
		},
		{
			name: "disabled_when_no_package_manager",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			env: map[string]string{
				env.ReleaseTrack:          "BETA",
				env.XGoogleTargetPlatform: "gcp",
			},
			want: false,
		},
		{
			name: "disabled_when_platform_is_not_gcp_or_gcf",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			env: map[string]string{
				env.PythonPackageManager:  "pip",
				env.ReleaseTrack:          "BETA",
				env.XGoogleTargetPlatform: "gae",
			},
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			if got := IsPipPyproject(ctx); got != tc.want {
				t.Errorf("IsPipPyproject() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsPyprojectEnabled(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  map[string]string
		want  bool
	}{
		{
			name: "enabled_with_only_pyproject_and_alpha_track",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{
				"X_GOOGLE_RELEASE_TRACK": "ALPHA",
			},
			want: true,
		},
		{
			name: "enabled_with_only_pyproject_and_beta_track",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{
				"X_GOOGLE_RELEASE_TRACK": "BETA",
			},
			want: true,
		},
		{
			name: "disabled_when_requirements_txt_exists",
			files: map[string]string{
				"requirements.txt": "flask",
				"pyproject.toml":   "[project]",
			},
			envs: map[string]string{
				"X_GOOGLE_RELEASE_TRACK": "ALPHA",
			},
			want: false,
		},
		{
			name: "enabled_on_GA_track_for_python_313",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.13.0",
			},
			want: true,
		},
		{
			name: "enabled_on_GA_track_for_python_314",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.14.0",
			},
			want: true,
		},
		{
			name: "enabled_on_GA_track_for_universal_22",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{},
			want: true,
		},
		{
			name: "enabled_on_GA_track_for_python_312",
			files: map[string]string{
				"pyproject.toml": "[project]",
			},
			envs: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.12.5",
			},
			want: true,
		},

		{
			name: "disabled_when_pyproject_toml_does_not_exist",
			files: map[string]string{
				"requirements.txt": "flask",
			},
			envs: map[string]string{
				"X_GOOGLE_RELEASE_TRACK": "ALPHA",
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)
			for key, value := range tc.envs {
				t.Setenv(key, value)
			}
			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			if gotEnabled := IsPyprojectEnabled(ctx); gotEnabled != tc.want {
				t.Errorf("IsPyprojectEnabled() = %v, want %v", gotEnabled, tc.want)
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
