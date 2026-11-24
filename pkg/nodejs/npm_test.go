// Copyright 2021 Google LLC
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

package nodejs

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"google3/security/safeopen/safeopen"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestRequestedNPMVersion(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON string
		want        string
		wantErr     bool
	}{
		{
			name:        "default is empty",
			packageJSON: `{}`,
			want:        "",
		},
		{
			name:        "engines.npm set",
			packageJSON: `{"engines": {"npm": "2.2.2"}}`,
			want:        "2.2.2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			dir := t.TempDir()
			var pjs *PackageJSON
			if tc.packageJSON != "" {
				if err := json.Unmarshal([]byte(tc.packageJSON), &pjs); err != nil {
					t.Errorf("failed to unmarshal package.json: %q, err: %v", tc.packageJSON, err)
				}
			}

			got, err := RequestedNPMVersion(pjs)
			if tc.wantErr == (err == nil) {
				t.Errorf("RequestedNPMVersion(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RequestedNPMVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestNPMInstallCommand(t *testing.T) {
	testCases := []struct {
		name           string
		npmVersion     string
		nodeVersion    string
		want           string
		targetPlatform string
		devSync        bool
	}{
		{
			name:       "8.3.1 should return ci",
			npmVersion: "8.3.1",
			want:       "ci",
		},
		{
			name:           "8.3.1 on GAE should return install for backwards compatibility",
			npmVersion:     "8.3.1",
			nodeVersion:    "10.24.1",
			want:           "install",
			targetPlatform: env.TargetPlatformAppEngine,
		},
		{
			name:           "8.3.1 on GCF should return install for backwards compatibility",
			npmVersion:     "8.3.1",
			nodeVersion:    "10.24.1",
			want:           "install",
			targetPlatform: env.TargetPlatformFunctions,
		},
		{
			name:       "5.7.0 should return install",
			npmVersion: "5.7.0",
			want:       "install",
		},
		{
			name:       "8.3.1 with dev sync should return install",
			npmVersion: "8.3.1",
			want:       "install",
			devSync:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) (string, error) { return tc.npmVersion, nil }
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcpbuildpack.Context) (string, error) { return tc.nodeVersion, nil }
			if tc.targetPlatform != "" {
				t.Setenv(env.XGoogleTargetPlatform, tc.targetPlatform)
			}
			if tc.devSync {
				t.Setenv(env.DevSync, "true")
				t.Setenv(env.XGoogleDevSyncActivated, "true")
			}

			got, err := NPMInstallCommand(nil)
			if err != nil {
				t.Fatalf("npm %v: NPMInstallCommand(nil) got error: %v", tc.npmVersion, err)
			}
			if got != tc.want {
				t.Errorf("npm %v: NPMInstallCommand(nil) = %q, want %q", tc.npmVersion, got, tc.want)
			}
		})
	}
}

func TestSupportsNPMPrune(t *testing.T) {
	testCases := []struct {
		version string
		want    bool
	}{
		{
			version: "8.3.1",
			want:    true,
		},
		{
			version: "5.7.0",
			want:    true,
		},
		{
			version: "5.0.1",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) (string, error) { return tc.version, nil }

			got, err := SupportsNPMPrune(nil)
			if err != nil {
				t.Errorf("npm %v: SupportsNPMPrune(nil) got error: %v", tc.version, err)
			}
			if got != tc.want {
				t.Errorf("npm %v: SupportsNPMPrune(nil) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}

func TestDetermineBuildCommands(t *testing.T) {
	testsCases := []struct {
		name                       string
		pjs                        string
		appHostingBuildScriptSet   bool
		appHostingBuildScriptValue string // ignored if `nextJsBuildScriptSet == false`
		nodeRunScriptSet           bool
		nodeRunScriptValue         string // ignored if `nodeRunScriptSet == false`
		targetPlatformSet          bool
		want                       []string
		wantIsCustomBuild          bool
	}{
		{
			name:             "no build",
			pjs:              "",
			nodeRunScriptSet: false,
			want:             []string{},
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "lint,clean,build",
			want:               []string{"npm run lint", "npm run clean", "npm run build"},
			wantIsCustomBuild:  true,
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS single value",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "lint",
			want:               []string{"npm run lint"},
			wantIsCustomBuild:  true,
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS trim whitespace",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "    	lint	,	 build",
			want:               []string{"npm run lint", "npm run build"},
			wantIsCustomBuild:  true,
		},
		{
			name: "build script",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			targetPlatformSet: true,
			want:              []string{"npm run build"},
		},
		{
			name: "build script runs regardless of target platform",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			targetPlatformSet: false,
			want:              []string{"npm run build"},
		},
		{
			name: "gcp-build script",
			pjs: `{
				"scripts": {
					"gcp-build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			want:              []string{"npm run gcp-build"},
			wantIsCustomBuild: true,
		},
		{
			name: "apphosting:build highest precedence",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build",
					"apphosting:build": "tsc --build"
				}
			}`,
			want:                       []string{"npm run apphosting:build"},
			wantIsCustomBuild:          true,
			appHostingBuildScriptSet:   true,
			appHostingBuildScriptValue: "next-js build",
			nodeRunScriptSet:           true,
			nodeRunScriptValue:         "from-env",
		},
		{
			name: "APPHOSTING_BUILD second precedence",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			appHostingBuildScriptSet:   true,
			appHostingBuildScriptValue: "next-js build",
			nodeRunScriptSet:           true,
			nodeRunScriptValue:         "from-env",
			want:                       []string{"next-js build"},
			wantIsCustomBuild:          true,
		},
		{
			name: "GOOGLE_NODE_RUN_SCRIPTS third precedence",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "from-env",
			want:               []string{"npm run from-env"},
			wantIsCustomBuild:  true,
		},
		{
			name: "gcp-build higher precedence than build",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			want:              []string{"npm run gcp-build"},
			wantIsCustomBuild: true,
		},
		{
			name: "empty gcp-build higher precedence than build and runs nothing",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "  "
				}
			}`,
			want:              []string{},
			wantIsCustomBuild: true,
		},
		{
			name: "empty build runs nothing",
			pjs: `{
				"scripts": {
					"build": ""
				}
			}`,
			want:              []string{},
			wantIsCustomBuild: false,
		},
		{
			name: "setting empty GOOGLE_NODE_RUN_SCRIPTS runs nothing",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "",
			want:               []string{},
			wantIsCustomBuild:  true,
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.appHostingBuildScriptSet {
				t.Setenv(AppHostingBuildEnv, tc.appHostingBuildScriptValue)
			}
			if tc.nodeRunScriptSet {
				t.Setenv(GoogleNodeRunScriptsEnv, tc.nodeRunScriptValue)
			}

			if tc.targetPlatformSet {
				t.Setenv(env.XGoogleTargetPlatform, env.TargetPlatformAppEngine)
			}

			var pjs *PackageJSON
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}
			got, isCustomBuild := DetermineBuildCommands(pjs, "npm")
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("DetermineBuildCommands() mismatch (-want +got):\n%s", diff)
			}

			if isCustomBuild != tc.wantIsCustomBuild {
				t.Errorf("DetermineBuildCommands() is custom build mismatch, got: %t, want: %t", isCustomBuild, tc.wantIsCustomBuild)
			}
		})
	}
}

func TestDefaultStartCommand(t *testing.T) {
	testsCases := []struct {
		name        string
		pjs         string
		hasServerJs bool
		want        []string
	}{
		{
			name: "no package.json",
			want: []string{"node", "index.js"},
		},
		{
			name: "no main or start",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			want: []string{"node", "index.js"},
		},
		{
			name: "main and start uses start",
			pjs: `{
				"main": "main.js",
				"scripts": {
					"start": "node main.js"
				}
			}`,
			want: []string{"npm", "run", "start"},
		},
		{
			name: "main module",
			pjs: `{
				"main": "main.js",
				"scripts": {
					"build": "node main.js"
				}
			}`,
			want: []string{"node", "main.js"},
		},
		{
			name: "main module and serverjs",
			pjs: `{
				"main": "main.js",
				"scripts": {
					"build": "node main.js"
				}
			}`,
			hasServerJs: true,
			want:        []string{"npm", "run", "start"},
		},
		{
			name: "angular app",
			pjs: `{
					"scripts": {
						"ng": "ng",
						"start": "ng serve",
						"build": "ng build",
						"watch": "ng build --watch --configuration development",
						"test": "ng test",
						"serve:ssr:my-angular-app": "node dist/my-angular-app/server/server.mjs"
					}
				}`,
			want: []string{"node", "dist/my-angular-app/server/server.mjs"},
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			var pjs *PackageJSON = nil
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}
			home := t.TempDir()
			if tc.hasServerJs {
				_, err := safeopen.CreateBeneath(home, "server.js")
				if err != nil {
					t.Fatalf("failed to create server.js: %v", err)
				}
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(home))

			got, err := DefaultStartCommand(ctx, pjs)
			if err != nil {
				t.Fatalf("DefaultStartCommand() got error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("DefaultStartCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestoreAndSaveModules(t *testing.T) {
	testCases := []struct {
		name       string
		capability any
		wantExec   bool
	}{
		{
			name:     "default_copier_executes_cp",
			wantExec: true,
		},
		{
			name:       "maker_copier_does_not_execute_cp",
			capability: &MakerNPMCacheCopier{},
			wantExec:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executed := false
			mockExec := func(cmd string, args ...string) *exec.Cmd {
				if cmd == "cp" {
					executed = true
				}
				return exec.Command("true")
			}

			opts := []gcpbuildpack.ContextOption{
				gcpbuildpack.WithExecCmd(mockExec),
			}
			if tc.capability != nil {
				opts = append(opts, gcpbuildpack.WithCapability(NPMCacheCopierCapability, tc.capability))
			}

			ctx := gcpbuildpack.NewContext(opts...)
			tmpDir := t.TempDir()
			src := filepath.Join(tmpDir, "src")
			dst := filepath.Join(tmpDir, "dst")

			if err := RestoreModules(ctx, src, dst); err != nil {
				t.Errorf("RestoreModules(%q, %q) returned unexpected error: %v", src, dst, err)
			}
			if executed != tc.wantExec {
				t.Errorf("RestoreModules(%q, %q) executed cp = %v, want %v", src, dst, executed, tc.wantExec)
			}

			executed = false
			if err := SaveModules(ctx, src, dst); err != nil {
				t.Errorf("SaveModules(%q, %q) returned unexpected error: %v", src, dst, err)
			}
			if executed != tc.wantExec {
				t.Errorf("SaveModules(%q, %q) executed cp = %v, want %v", src, dst, executed, tc.wantExec)
			}
		})
	}
}

func TestWatchEntrypointFile(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON *PackageJSON
		files       []string
		want        string
	}{
		{
			name: "main",
			packageJSON: &PackageJSON{
				Main: "main.js",
			},
			want: "main.js",
		},
		{
			name:        "server_js",
			packageJSON: &PackageJSON{},
			files:       []string{"server.js", "index.js"},
			want:        "server.js",
		},
		{
			name:        "index_js",
			packageJSON: &PackageJSON{},
			files:       []string{"index.js"},
			want:        "index.js",
		},
		{
			name:        "default",
			packageJSON: &PackageJSON{},
			want:        "index.js",
		},
		{
			name:        "nil_package_json_with_server_js",
			packageJSON: nil,
			files:       []string{"server.js"},
			want:        "server.js",
		},
		{
			name:        "nil_package_json_without_server_js",
			packageJSON: nil,
			files:       []string{},
			want:        "index.js",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(testDir, f), []byte(""), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", f, err)
				}
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(testDir))
			got, err := watchEntrypointFile(ctx, tc.packageJSON)
			if err != nil {
				t.Fatalf("watchEntrypointFile got error: %v", err)
			}
			if got != tc.want {
				t.Errorf("watchEntrypointFile() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDevSyncEntrypoint(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON *PackageJSON
		files       []string
		pkgManager  string
		want        []string
	}{
		{
			name: "dev script",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{"dev": "node server.js"},
			},
			pkgManager: "npm",
			want:       []string{"npm", "run", "dev"},
		},
		{
			name: "watch server.js",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{},
			},
			files:      []string{"server.js", "index.js"},
			pkgManager: "npm",
			want:       []string{"node", "--watch", "server.js"},
		},
		{
			name: "watch index.js",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{},
			},
			files:      []string{"index.js"},
			pkgManager: "npm",
			want:       []string{"node", "--watch", "index.js"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := t.TempDir()
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(testDir, f), []byte(""), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", f, err)
				}
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(testDir))
			got, err := DevSyncEntrypoint(ctx, tc.packageJSON, tc.pkgManager)
			if err != nil {
				t.Fatalf("DevSyncEntrypoint got error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("DevSyncEntrypoint() returned unexpected difference (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDetermineBuildCommandsWebFrameworks(t *testing.T) {
	testCases := []struct {
		name        string
		pkgTool     string
		rootPkgJSON string
		files       map[string]string
		envs        map[string]string
		wantCmds    []string
		wantCustom  bool
	}{
		{
			name:        "gate_off_is_strict_noop_even_with_monorepo_env",
			pkgTool:     "npm",
			rootPkgJSON: `{"scripts":{"build":"next build"}}`,
			envs: map[string]string{
				EnvWebFrameworks:     "false",
				EnvWebAdapterActive:  "true",
				EnvMonorepoCommand:   "turbo",
				EnvMonorepoProject:   "web",
				EnvMonorepoBuildArgs: "--filter=web,--env-mode=loose",
				env.Buildable:        "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "fah_is_strict_noop_even_with_gate_on_and_monorepo_env",
			pkgTool:     "npm",
			rootPkgJSON: `{"scripts":{"build":"next build"}}`,
			envs: map[string]string{
				EnvWebFrameworks:          "true",
				env.ReleaseTrack:          "ALPHA",
				env.XGoogleTargetPlatform: "fah",
				EnvWebAdapterActive:       "true",
				EnvMonorepoCommand:        "turbo",
				EnvMonorepoProject:        "web",
				EnvMonorepoBuildArgs:      "--filter=web,--env-mode=loose",
				env.Buildable:             "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "flag_on_but_adapter_inactive_keeps_root_build_with_monorepo_env",
			pkgTool:     "npm",
			rootPkgJSON: `{"scripts":{"build":"turbo run build"}}`,
			envs: map[string]string{
				EnvWebFrameworks:     "true",
				env.ReleaseTrack:     "ALPHA",
				EnvWebAdapterActive:  "",
				EnvMonorepoCommand:   "turbo",
				EnvMonorepoProject:   "web",
				EnvMonorepoBuildArgs: "--filter=web,--env-mode=loose",
				env.Buildable:        "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "flag_on_but_adapter_inactive_keeps_root_build_for_plain_workspace",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"],"scripts":{"build":"npm run build --workspaces"}}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web","scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "false",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "turbo_monorepo_with_gate_on_and_adapter_active",
			pkgTool:     "pnpm",
			rootPkgJSON: `{"private":true}`,
			envs: map[string]string{
				EnvWebFrameworks:     "true",
				env.ReleaseTrack:     "ALPHA",
				EnvWebAdapterActive:  "true",
				EnvMonorepoCommand:   "turbo",
				EnvMonorepoProject:   "web",
				EnvMonorepoBuildArgs: "--filter=web,--env-mode=loose",
				env.Buildable:        "apps/web",
			},
			wantCmds: []string{"node_modules/.bin/turbo run build --filter=web --env-mode=loose"},
		},
		{
			name:        "nx_monorepo_with_gate_on_and_adapter_active",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true}`,
			envs: map[string]string{
				EnvWebFrameworks:     "true",
				env.ReleaseTrack:     "ALPHA",
				EnvWebAdapterActive:  "true",
				EnvMonorepoCommand:   "nx",
				EnvMonorepoProject:   "web",
				EnvMonorepoBuildArgs: "--project=web",
				env.Buildable:        "apps/web",
			},
			wantCmds: []string{"node_modules/.bin/nx run web:build"},
		},
		{
			name:        "root_gcp_build_takes_precedence_over_monorepo",
			pkgTool:     "npm",
			rootPkgJSON: `{"scripts":{"gcp-build":"custom-build"}}`,
			envs: map[string]string{
				EnvWebFrameworks:     "true",
				env.ReleaseTrack:     "ALPHA",
				EnvWebAdapterActive:  "true",
				EnvMonorepoCommand:   "turbo",
				EnvMonorepoProject:   "web",
				EnvMonorepoBuildArgs: "--filter=web,--env-mode=loose",
			},
			wantCmds:   []string{"npm run gcp-build"},
			wantCustom: true,
		},
		{
			name:        "plain_npm_workspace_targeting",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"]}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web","scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"npm run build -w apps/web"},
		},
		{
			name:        "plain_pnpm_workspace_targeting",
			pkgTool:     "pnpm",
			rootPkgJSON: `{"private":true}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web","scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"pnpm --filter ./apps/web run build"},
		},
		{
			name:        "plain_yarn_workspace_targeting",
			pkgTool:     "yarn",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"]}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web","scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"yarn workspace @repo/web run build"},
		},
		{
			name:        "plain_bun_workspace_targeting",
			pkgTool:     "bun",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"]}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web","scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"bun run --filter ./apps/web build"},
		},
		{
			name:        "plain_bun_workspace_targeting_without_name",
			pkgTool:     "bun",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"]}`,
			files: map[string]string{
				"apps/web/package.json": `{"scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"bun run --filter ./apps/web build"},
		},
		{
			name:        "plain_workspace_malformed_app_package_json_falls_back_to_root_build",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"],"scripts":{"build":"npm run build:all"}}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web",`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "plain_workspace_root_only_build_script_falls_back_to_root_build",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true,"workspaces":["apps/*"],"scripts":{"build":"npm run build:all"}}`,
			files: map[string]string{
				"apps/web/package.json": `{"name":"@repo/web"}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
		{
			name:        "turbo_stood_down_missing_name_falls_back_to_root_build",
			pkgTool:     "npm",
			rootPkgJSON: `{"private":true,"scripts":{"build":"turbo run build"}}`,
			files: map[string]string{
				"turbo.json":            `{}`,
				"apps/web/package.json": `{"scripts":{"build":"next build"}}`,
			},
			envs: map[string]string{
				EnvWebFrameworks:    "true",
				env.ReleaseTrack:    "ALPHA",
				EnvWebAdapterActive: "true",
				env.Buildable:       "apps/web",
			},
			wantCmds: []string{"npm run build"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := t.TempDir()
			origWorkDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("os.Getwd() failed: %v", err)
			}
			if err := os.Chdir(rootDir); err != nil {
				t.Fatalf("os.Chdir(%q) failed: %v", rootDir, err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(origWorkDir)
			})

			for rel, content := range tc.files {
				full := filepath.Join(rootDir, rel)
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatalf("os.MkdirAll(%q) failed: %v", filepath.Dir(full), err)
				}
				if err := os.WriteFile(full, []byte(content), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q) failed: %v", full, err)
				}
			}

			t.Setenv(EnvWebFrameworks, "")
			t.Setenv(EnvWebAdapterActive, "")
			t.Setenv(env.XGoogleTargetPlatform, "")
			t.Setenv(EnvMonorepoCommand, "")
			t.Setenv(EnvMonorepoProject, "")
			t.Setenv(EnvMonorepoBuildArgs, "")
			t.Setenv(env.Buildable, "")
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			var pkgJSON *PackageJSON
			if tc.rootPkgJSON != "" {
				if err := json.Unmarshal([]byte(tc.rootPkgJSON), &pkgJSON); err != nil {
					t.Fatalf("json.Unmarshal(%q) failed: %v", tc.rootPkgJSON, err)
				}
			}

			gotCmds, gotCustom := DetermineBuildCommands(pkgJSON, tc.pkgTool)
			if diff := cmp.Diff(tc.wantCmds, gotCmds); diff != "" {
				t.Errorf("DetermineBuildCommands(%v, %q) cmds diff (-want +got):\n%s", pkgJSON, tc.pkgTool, diff)
			}
			if gotCustom != tc.wantCustom {
				t.Errorf("DetermineBuildCommands(%v, %q) isCustomBuild = %v, want %v", pkgJSON, tc.pkgTool, gotCustom, tc.wantCustom)
			}
		})
	}
}
