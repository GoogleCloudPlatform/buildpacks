// Copyright 2025 Google LLC
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

package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "without package without yarn",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
		{
			name: "with package without yarn",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			want: 100,
		},
		{
			name: "without package with yarn",
			files: map[string]string{
				"index.js":  "",
				"yarn.lock": "",
			},
			want: 100,
		},
		{
			name: "with yarn and package",
			files: map[string]string{
				"index.js":     "",
				"yarn.lock":    "",
				"package.json": "",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestYarn1InstallModulesWebAdapterGateOffVsOn(t *testing.T) {
	testCases := []struct {
		name                 string
		envs                 map[string]string
		mocks                []*mockprocess.Mock
		wantYarnModulesLayer bool
	}{
		{
			name: "gate_off_uses_yarn_modules_layer_and_skips_plain_build",
			envs: map[string]string{
				nodejs.EnvWebFrameworks:    "",
				nodejs.EnvWebAdapterActive: "",
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`node -v`, mockprocess.WithStdout("v22.0.0")),
				mockprocess.New(`yarn install --non-interactive --prefer-offline --modules-folder=yarn_modules/node_modules`, mockprocess.WithStdout("installed")),
			},
			wantYarnModulesLayer: true,
		},
		{
			name: "gate_off_even_with_adapter_active_env_uses_yarn_modules_layer",
			envs: map[string]string{
				nodejs.EnvWebFrameworks:    "",
				nodejs.EnvWebAdapterActive: "true",
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`node -v`, mockprocess.WithStdout("v22.0.0")),
				mockprocess.New(`yarn install --non-interactive --prefer-offline --modules-folder=yarn_modules/node_modules`, mockprocess.WithStdout("installed")),
			},
			wantYarnModulesLayer: true,
		},
		{
			name: "web_adapter_active_installs_into_workspace_and_runs_build_without_yarn_modules_layer",
			envs: map[string]string{
				nodejs.EnvWebFrameworks:    "true",
				"X_GOOGLE_RELEASE_TRACK":   "ALPHA",
				nodejs.EnvWebAdapterActive: "true",
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`node -v`, mockprocess.WithStdout("v22.0.0")),
				mockprocess.New(`yarn install --non-interactive --prefer-offline --modules-folder=.*/node_modules --production=false`, mockprocess.WithStdout("installed")),
				mockprocess.New(`yarn run build`, mockprocess.WithStdout("built")),
				mockprocess.New(`yarn install --ignore-scripts --prefer-offline --production=true --modules-folder=.*/node_modules`, mockprocess.WithStdout("pruned")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			buildpackDir := t.TempDir()

			execCmd, err := mockprocess.NewExecCmd(tc.mocks...)
			if err != nil {
				t.Fatalf("mockprocess.NewExecCmd() failed: %v", err)
			}

			origWorkDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("os.Getwd() failed: %v", err)
			}
			if err := os.Chdir(buildpackDir); err != nil {
				t.Fatalf("os.Chdir(%q) failed: %v", buildpackDir, err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(origWorkDir)
			})

			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			pkgJSONRaw := `{"scripts":{"build":"next build"},"devDependencies":{"typescript":"^5.0.0"}}`
			if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(pkgJSONRaw), 0644); err != nil {
				t.Fatalf("os.WriteFile(package.json) failed: %v", err)
			}
			var pkgJSON nodejs.PackageJSON
			if err := json.Unmarshal([]byte(pkgJSONRaw), &pkgJSON); err != nil {
				t.Fatalf("json.Unmarshal(package.json) failed: %v", err)
			}

			ctx := gcp.NewContext(
				gcp.WithApplicationRoot(appDir),
				gcp.WithBuildpackRoot(buildpackDir),
				gcp.WithExecCmd(execCmd),
			)
			if err := yarn1InstallModules(ctx, &pkgJSON); err != nil {
				t.Fatalf("yarn1InstallModules() unexpected error: %v", err)
			}
			_, statErr := os.Stat(filepath.Join(buildpackDir, "yarn_modules"))
			if gotLayer := statErr == nil; gotLayer != tc.wantYarnModulesLayer {
				t.Errorf("yarn1InstallModules() created yarn_modules layer = %v, want %v (stat err: %v)", gotLayer, tc.wantYarnModulesLayer, statErr)
			}
		})
	}
}

func TestYarn2InstallModulesWebAdapter(t *testing.T) {
	appDir := t.TempDir()
	buildpackDir := t.TempDir()

	execCmd, err := mockprocess.NewExecCmd(
		mockprocess.New(`yarn install --immutable`, mockprocess.WithStdout("installed")),
		mockprocess.New(`yarn run build`, mockprocess.WithStdout("built")),
	)
	if err != nil {
		t.Fatalf("mockprocess.NewExecCmd() failed: %v", err)
	}

	origWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}
	if err := os.Chdir(appDir); err != nil {
		t.Fatalf("os.Chdir(%q) failed: %v", appDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWorkDir)
	})

	t.Setenv(nodejs.EnvWebFrameworks, "true")
	t.Setenv("X_GOOGLE_RELEASE_TRACK", "ALPHA")
	t.Setenv(nodejs.EnvWebAdapterActive, "true")

	pkgJSONRaw := `{"scripts":{"build":"next build"}}`
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(pkgJSONRaw), 0644); err != nil {
		t.Fatalf("os.WriteFile(package.json) failed: %v", err)
	}
	var pkgJSON nodejs.PackageJSON
	if err := json.Unmarshal([]byte(pkgJSONRaw), &pkgJSON); err != nil {
		t.Fatalf("json.Unmarshal(package.json) failed: %v", err)
	}

	ctx := gcp.NewContext(
		gcp.WithApplicationRoot(appDir),
		gcp.WithBuildpackRoot(buildpackDir),
		gcp.WithExecCmd(execCmd),
	)
	if err := yarn2InstallModules(ctx, &pkgJSON); err != nil {
		t.Fatalf("yarn2InstallModules() unexpected error: %v", err)
	}
}
