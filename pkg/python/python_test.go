// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package python

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/buildpacks/libcnb/v2"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestRuntimeVersion(t *testing.T) {
	testCases := []struct {
		name           string
		version        string
		runtimeVersion string
		versionFile    string
		want           string
		wantErr        bool
		wantMatch      bool
		wantMatchErr   bool
		stackID        string
	}{
		{
			name: "default_to_latest_for_default_stack_ubuntu2404_is_default_for_unit_tests",
			want: "3.14.*",
		},
		{
			name:    "default_to_latest_for_stack_ubuntu2404",
			stackID: "google.24.full",
			want:    "3.14.*",
		},
		{
			name:    "default_to_latest_for_stack_ubuntu1804",
			stackID: "google.gae.18",
			want:    "3.9.*",
		},
		{
			name:    "version_from_GOOGLE_PYTHON_VERSION",
			version: "3.8.0",
			want:    "3.8.0",
		},
		{
			name:           "version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.8.0",
			want:           "3.8.0",
		},
		{
			name:           "GOOGLE_PYTHON_VERSION_take_precedence_over_GOOGLE_RUNTIME_VERSION",
			version:        "3.8.0",
			runtimeVersion: "3.8.1",
			want:           "3.8.0",
		},
		{
			name:        "version_from_.python-version_file",
			versionFile: "3.8.0",
			want:        "3.8.0",
		},
		{
			name:        "empty_.python-version_file",
			versionFile: " ",
			wantErr:     true,
		},
		{
			name:           "GOOGLE_RUNTIME_VERSION_take_precedence_over_.python-version",
			runtimeVersion: "3.8.0",
			versionFile:    "3.8.1",
			want:           "3.8.0",
		},
		{
			name:           "version_above_3.13.0_through_runtime_version",
			runtimeVersion: "3.13.1",
			want:           "3.13.1",
		},
		{
			name:    "version_below_3.13.0",
			version: "3.12.1",
			want:    "3.12.1",
		},
		{
			name:    "version_with_prerelease",
			version: "3.14.0a1",
			want:    "3.14.0a1",
		},
		{
			name:    "version_with_RC",
			version: "3.13.0rc1",
			want:    "3.13.0rc1",
		},
		{
			name:    "No_version_but_stackID_is_google.22",
			stackID: "google.22",
			want:    "3.13.*",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))
			if tc.stackID != "" {
				ctx = gcp.NewContext(gcp.WithApplicationRoot(dir), gcp.WithStackID(tc.stackID))
			}

			if tc.version != "" {
				t.Setenv("GOOGLE_PYTHON_VERSION", tc.version)
			}
			if tc.runtimeVersion != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.runtimeVersion)
			}
			if tc.versionFile != "" {
				versionFile := filepath.Join(dir, ".python-version")
				if err := os.WriteFile(versionFile, []byte(tc.versionFile), os.FileMode(0744)); err != nil {
					t.Fatalf("writing file %q: %v", versionFile, err)
				}
			}

			got, err := RuntimeVersion(ctx, dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("RuntimeVersion(ctx, %q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RuntimeVersion(ctx, %q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestVersionMatchesSemver(t *testing.T) {
	testCases := []struct {
		name         string
		versionRange string
		version      string
		want         bool
		wantErr      bool
	}{
		{
			name:         "version_matches_semver_range",
			versionRange: ">=3.13.0",
			version:      "3.13.1",
			want:         true,
		},
		{
			name:         "version_does_not_match_semver_range",
			versionRange: ">=3.13.0",
			version:      "3.12.1",
			want:         false,
		},
		{
			name:         "invalid_version_range",
			versionRange: "3.13.0",
			version:      "3.12.1",
			want:         false,
		},
		{
			name:         "invalid_version",
			versionRange: ">=3.13.0",
			version:      "3.12.1a",
			wantErr:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			got, err := versionMatchesSemver(ctx, tc.versionRange, tc.version)
			if tc.wantErr == (err == nil) {
				t.Errorf("versionMatchesSemver(ctx, %q, %q) got error: %v, want err? %t", tc.versionRange, tc.version, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("versionMatchesSemver(ctx, %q, %q) = %t, want %t", tc.versionRange, tc.version, got, tc.want)
			}
		})
	}
}

func TestSupportSmartDefaultEntrypoint(t *testing.T) {
	testCases := []struct {
		name           string
		version        string
		runtimeVersion string
		versionFile    string
		stackID        string
		want           bool
		wantErr        bool
	}{
		{
			name: "default_to_latest_for_default_stack_ubuntu2204_is_default_for_unit_tests",
			want: true,
		},
		{
			name:    "default_to_latest_for_stack_ubuntu1804",
			stackID: "google.gae.18",
			want:    false,
		},
		{
			name:    "supported_version_from_GOOGLE_PYTHON_VERSION",
			version: "3.14.0",
			want:    true,
		},
		{
			name:    "unsupported_version_from_GOOGLE_PYTHON_VERSION",
			version: "3.8.0",
			want:    false,
		},
		{
			name:           "unsupported_version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.8.0",
			want:           false,
		},
		{
			name:           "supported_version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.13.8",
			want:           true,
		},
		{
			name:        "empty_.python-version_file",
			versionFile: " ",
			wantErr:     true,
		},
		{
			name:    "version_above_3.13.0",
			version: "3.13.1",
			want:    true,
		},
		{
			name:    "version_below_3.13.0",
			version: "3.12.1",
			want:    false,
		},
		{
			name:    "version_with_prerelease",
			version: "3.14.0a1",
			wantErr: true,
			want:    false, // We don't support prerelease versions. Modify once we add support for prerelease versions.
		},
		{
			name:    "version_with_RC",
			version: "3.14.0rc1",
			wantErr: false,
			want:    true, // Support for RC versions added.
		},
		{
			name:    "No_version_but_stackID_is_google.22",
			stackID: "google.22",
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))
			if tc.stackID != "" {
				ctx = gcp.NewContext(gcp.WithApplicationRoot(dir), gcp.WithStackID(tc.stackID))
			}

			if tc.version != "" {
				t.Setenv("GOOGLE_PYTHON_VERSION", tc.version)
			}
			if tc.runtimeVersion != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.runtimeVersion)
			}
			if tc.versionFile != "" {
				versionFile := filepath.Join(dir, ".python-version")
				if err := os.WriteFile(versionFile, []byte(tc.versionFile), os.FileMode(0744)); err != nil {
					t.Fatalf("writing file %q: %v", versionFile, err)
				}
			}

			boolGot, err := SupportsSmartDefaultEntrypoint(ctx)
			if tc.wantErr == (err == nil) {
				t.Errorf("SupportsSmartDefaultEntrypoint(ctx, %q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if boolGot != tc.want {
				t.Errorf("SupportsSmartDefaultEntrypoint(ctx, %q) = %t, want %t", dir, boolGot, tc.want)
			}
		})
	}
}

func TestIsUVDefaultPackageManagerForRequirements(t *testing.T) {
	testCases := []struct {
		name  string
		envs  map[string]string
		files map[string]string
		want  bool
	}{
		{name: "env_py_313", envs: map[string]string{"GOOGLE_PYTHON_VERSION": "3.13.0"}, want: false},
		{name: "env_py_313_9", envs: map[string]string{"GOOGLE_PYTHON_VERSION": "3.13.9"}, want: false},
		{name: "env_py_314_0", envs: map[string]string{"GOOGLE_PYTHON_VERSION": "3.14.0"}, want: true},
		{name: "env_py_314_1", envs: map[string]string{"GOOGLE_PYTHON_VERSION": "3.14.1"}, want: true},
		{name: "env_py_315", envs: map[string]string{"GOOGLE_PYTHON_VERSION": "3.15.0"}, want: true},
		{name: "env_runtime_313", envs: map[string]string{"GOOGLE_RUNTIME_VERSION": "3.13.0"}, want: false},
		{name: "env_runtime_313_9", envs: map[string]string{"GOOGLE_RUNTIME_VERSION": "3.13.9"}, want: false},
		{name: "env_runtime_314_0", envs: map[string]string{"GOOGLE_RUNTIME_VERSION": "3.14.0"}, want: true},
		{name: "env_runtime_314_1", envs: map[string]string{"GOOGLE_RUNTIME_VERSION": "3.14.1"}, want: true},
		{name: "file_py_313", files: map[string]string{".python-version": "3.13.0\n"}, want: false},
		{name: "file_py_314", files: map[string]string{".python-version": "3.14.0\n"}, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := setupTest(t, tc.files)

			for key, value := range tc.envs {
				t.Setenv(key, value)
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))

			if got := isUVDefaultPackageManagerForRequirements(ctx); got != tc.want {
				t.Errorf("isUVDefaultPackageManagerForRequirements() with envs %v, files %v got %v, want %v", tc.envs, tc.files, got, tc.want)
			}
		})
	}
}

func TestPrepareDependenciesLayer(t *testing.T) {
	const defaultPythonVersion = "3.10.5"
	const defaultInstaller = "pip"

	testCases := []struct {
		name           string
		twoRun         bool
		files1         map[string]string
		pythonVersion1 string
		installerName1 string
		reqs           []string
		files2         map[string]string
		pythonVersion2 string
		installerName2 string
		wantProceed    bool
		wantErr        bool
	}{
		{
			name:           "no_requirements",
			twoRun:         false,
			pythonVersion1: defaultPythonVersion,
			reqs:           []string{},
			wantProceed:    false,
			wantErr:        false,
		},
		{
			name:           "cache_miss_on_first_run",
			twoRun:         false,
			files1:         map[string]string{"reqs.txt": "flask"},
			pythonVersion1: defaultPythonVersion,
			reqs:           []string{"reqs.txt"},
			wantProceed:    true,
			wantErr:        false,
		},
		{
			name:           "cache_hit_on_second_run",
			twoRun:         true,
			files1:         map[string]string{"reqs.txt": "flask"},
			pythonVersion1: defaultPythonVersion,
			reqs:           []string{"reqs.txt"},
			wantProceed:    false,
			wantErr:        false,
		},
		{
			name:           "cache_invalidation_by_python_version",
			twoRun:         true,
			files1:         map[string]string{"reqs.txt": "flask"},
			reqs:           []string{"reqs.txt"},
			pythonVersion1: "3.10.5",
			pythonVersion2: "3.11.1",
			wantProceed:    true,
			wantErr:        false,
		},
		{
			name:           "cache_invalidation_by_installer_name",
			twoRun:         true,
			files1:         map[string]string{"reqs.txt": "flask"},
			reqs:           []string{"reqs.txt"},
			pythonVersion1: defaultPythonVersion,
			installerName1: "pip",
			installerName2: "uv",
			wantProceed:    true,
			wantErr:        false,
		},
		{
			name:           "cache_invalidation_by_file_content",
			twoRun:         true,
			files1:         map[string]string{"reqs.txt": "flask"},
			pythonVersion1: defaultPythonVersion,
			reqs:           []string{"reqs.txt"},
			files2:         map[string]string{"reqs.txt": "django"},
			wantProceed:    true,
			wantErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			writeFakePythonScript(t, binDir, tc.pythonVersion1)
			oldPath := os.Getenv("PATH")
			t.Setenv("PATH", binDir+string(filepath.ListSeparator)+oldPath)

			appDir := setupTest(t, tc.files1)
			l := &libcnb.Layer{Name: "test", Path: t.TempDir(), Metadata: map[string]any{}}

			var reqPaths []string
			for _, r := range tc.reqs {
				reqPaths = append(reqPaths, filepath.Join(appDir, r))
			}

			installerName1 := tc.installerName1
			if installerName1 == "" {
				installerName1 = defaultInstaller
			}

			var proceed bool
			var err error
			ctx1 := gcp.NewContext(gcp.WithApplicationRoot(appDir))

			if !tc.twoRun {
				proceed, err = prepareDependenciesLayer(ctx1, l, installerName1, reqPaths...)
			} else {
				// Run 1 (Cache Miss)
				proceed1, err1 := prepareDependenciesLayer(ctx1, l, installerName1, reqPaths...)
				if err1 != nil {
					t.Fatalf("Run 1 (cache miss) failed: %v", err1)
				}
				if !proceed1 {
					t.Fatal("Run 1 (cache miss) should have returned proceed=true")
				}

				// Setup for Run 2
				if tc.files2 != nil {
					for name, content := range tc.files2 {
						if err := os.WriteFile(filepath.Join(appDir, name), []byte(content), 0644); err != nil {
							t.Fatalf("Failed to rewrite file for run 2: %v", err)
						}
					}
				}

				installerName2 := tc.installerName2
				if installerName2 == "" {
					installerName2 = installerName1
				}

				pythonVersion2 := tc.pythonVersion1
				if tc.pythonVersion2 != "" {
					pythonVersion2 = tc.pythonVersion2
				}
				writeFakePythonScript(t, binDir, pythonVersion2)

				ctx2 := gcp.NewContext(gcp.WithApplicationRoot(appDir))

				proceed, err = prepareDependenciesLayer(ctx2, l, installerName2, reqPaths...)
			}

			if (err != nil) != tc.wantErr {
				t.Fatalf("prepareDependenciesLayer() error = %v, wantErr %v", err, tc.wantErr)
			}
			if proceed != tc.wantProceed {
				t.Errorf("prepareDependenciesLayer() proceed = %v, want %v", proceed, tc.wantProceed)
			}
		})
	}
}

// writeFakePythonScript is a helper to create fake python3 executable
func writeFakePythonScript(t *testing.T, binDir, version string) string {
	t.Helper()
	fakePython := filepath.Join(binDir, "python3")
	content := fmt.Sprintf("#!/bin/sh\necho 'Python %s'\n", version)
	if err := os.WriteFile(fakePython, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to write fake python script: %v", err)
	}
	return fakePython
}

func TestParseExecPrefix(t *testing.T) {
	testCases := []struct {
		sysConfig string
		want      string
		wantErr   bool
	}{
		{
			sysConfig: "",
			want:      "",
			wantErr:   true,
		},
		{
			sysConfig: `installed_base = "/layers/google.python.runtime/python"`,
			want:      "",
			wantErr:   true,
		},
		{
			sysConfig: `
exec_prefix = "/opt/python3.11"
installed_base = "/layers/google.python.runtime/python"
			`,
			want: "/opt/python3.11",
		},
		{
			sysConfig: `
exec_prefix = "/opt/python3.9"
installed_base = "/layers/google.python.runtime/python"
			`,
			want: "/opt/python3.9",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.sysConfig, func(t *testing.T) {
			got, err := parseExecPrefix(tc.sysConfig)
			if (err == nil) == tc.wantErr {
				t.Errorf("parseExecPrefix() got err: %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("parseExecPrefix(%q) = %q, want %q", tc.sysConfig, got, tc.want)
			}
		})
	}
}

func TestAdaptEntrypoint(t *testing.T) {
	testCases := []struct {
		name      string
		cmd       []string
		scriptCmd []string
		want      []string
	}{
		{
			name: "python",
			cmd:  []string{"python", "main.py"},
			want: []string{"python", "main.py"},
		},
		{
			name: "python3",
			cmd:  []string{"python3", "main.py"},
			want: []string{"python3", "main.py"},
		},
		{
			name: "gunicorn",
			cmd:  []string{"gunicorn", "-b", ":8080", "main:app"},
			want: []string{"python3", "lib/bin/gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "uvicorn",
			cmd:  []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
			want: []string{"python3", "lib/bin/uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "streamlit",
			cmd:  []string{"streamlit", "run", "main.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
			want: []string{"python3", "lib/bin/streamlit", "run", "main.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
		},
		{
			name: "adk",
			cmd:  []string{"adk", "api_server", "--port", "8080", "--host", "0.0.0.0"},
			want: []string{"python3", "lib/bin/adk", "api_server", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "uv",
			cmd:  []string{"uv", "run", "main.py"},
			want: []string{"python3", "lib/bin/uv", "run", "main.py"},
		},
		{
			name:      "script_cmd_priority",
			cmd:       []string{"uv", "run", "foo"},
			scriptCmd: []string{"foo"},
			want:      []string{"python3", "lib/bin/foo"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithCapability(EntrypointAdapterCapability, &MakerEntrypointAdapter{}))
			got, err := AdaptEntrypoint(ctx, tc.cmd, tc.scriptCmd)
			if err != nil {
				t.Fatalf("AdaptEntrypoint(%v) failed: %v", tc.cmd, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("AdaptEntrypoint(%v) returned diff (-want +got):\n%s", tc.cmd, diff)
			}
		})
	}
}

func TestBaseuvPipInstallArgs(t *testing.T) {
	testCases := []struct {
		name string
		req  string
		want []string
	}{
		{
			name: "install_from_current_directory",
			req:  ".",
			want: []string{"uv", "pip", "install", ".", "--reinstall", "--link-mode=copy"},
		},
		{
			name: "install_from_requirements_txt",
			req:  "requirements.txt",
			want: []string{"uv", "pip", "install", "-r", "requirements.txt", "--reinstall", "--link-mode=copy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := baseuvPipInstallArgs(tc.req)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("baseuvPipInstallArgs(%q) returned diff (-want +got):\n%s", tc.req, diff)
			}
		})
	}
}
