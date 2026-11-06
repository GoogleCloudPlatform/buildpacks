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

package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	bmd "github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb/v2"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  int
	}{
		{
			name: "gate_off_opts_out",
			files: map[string]string{
				"next.config.js": "",
				"package.json":   `{"dependencies": {"next": "^16.4.0"}}`,
			},
			want: 100,
		},
		{
			name: "fah_platform_opts_out_even_with_gate_on",
			files: map[string]string{
				"next.config.js": "",
				"package.json":   `{"dependencies": {"next": "^16.4.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 100,
		},
		{
			name: "dev_mode_opts_out_even_with_gate_on",
			files: map[string]string{
				"next.config.js": "",
				"package.json":   `{"dependencies": {"next": "^16.4.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_DEVMODE=true"},
			want: 100,
		},
		{
			name: "gae_platform_opts_out",
			files: map[string]string{
				"next.config.js": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 100,
		},
		{
			name: "opts_in_with_next_config_js",
			files: map[string]string{
				"next.config.js": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_config_mjs",
			files: map[string]string{
				"next.config.mjs": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_config_cjs",
			files: map[string]string{
				"next.config.cjs": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_config_ts",
			files: map[string]string{
				"next.config.ts": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_config_mts",
			files: map[string]string{
				"next.config.mts": "",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_dependency_no_lockfile",
			files: map[string]string{
				"package.json": `{"dependencies": {"next": "^16.4.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_dev_dependency",
			files: map[string]string{
				"package.json": `{"devDependencies": {"next": "^16.4.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_google_buildable",
			files: map[string]string{
				"apps/web/next.config.ts": "",
				"package.json":            `{"private": true}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			want: 0,
		},
		{
			name: "opts_out_when_not_nextjs",
			files: map[string]string{
				"package.json": `{"dependencies": {"express": "^4.18.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 100,
		},
		{
			name: "opts_out_in_non_nx_monorepo_when_app_package_json_has_no_next",
			files: map[string]string{
				"package.json":              `{"devDependencies": {"next": "16.4.3"}}`,
				"services/api/package.json": `{"dependencies": {"express": "4.19.0"}, "scripts": {"build": "tsc"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=services/api"},
			want: 100,
		},
		{
			name: "opts_in_in_nx_monorepo_when_next_in_root_package_json",
			files: map[string]string{
				"nx.json":               `{}`,
				"package.json":          `{"dependencies": {"next": "16.4.3"}}`,
				"apps/web/package.json": `{"name": "web"}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			want: 0,
		},
		{
			name: "opts_in_in_monorepo_when_app_has_no_package_json_and_root_has_next",
			files: map[string]string{
				"package.json":        `{"dependencies": {"next": "16.4.3"}}`,
				"apps/web/src/app.ts": "console.log('hi')",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			want: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name             string
		files            map[string]string
		envs             []string
		mocks            []*mockprocess.Mock
		wantInstallCmd   string
		wantMetadata     map[bmd.MetadataID]bmd.MetadataValue
		wantLogSubstring string
	}{
		{
			name: "happy_path_default_adapter_version",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "^16.4.0"}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {"version": "16.4.3"}
					}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
				bmd.AdapterName:      "@google-cloud/adapter-nextjs",
				bmd.AdapterVersion:   "0.1.0",
			},
		},
		{
			name: "happy_path_bun_lock_and_custom_adapter_tgz",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start -p $PORT"},
					"dependencies": {"next": "^16.3.5"}
				}`,
				"bun.lock": `{
					"lockfileVersion": 1,
					"packages": {
						"next": ["next@16.3.5", "", {}, "sha512-xyz"],
					},
				}`,
			},
			envs: []string{
				"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA",
				"GOOGLE_NEXTJS_ADAPTER_VERSION=/tmp/google-cloud-adapter-nextjs-0.1.0.tgz",
			},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter /tmp/google-cloud-adapter-nextjs-0.1.0.tgz",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.3.5",
				bmd.AdapterName:      "@google-cloud/adapter-nextjs",
				bmd.AdapterVersion:   "/tmp/google-cloud-adapter-nextjs-0.1.0.tgz",
			},
		},
		{
			name: "happy_path_no_lockfile_uses_lowest_range_version",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": ">=16.3.5 <17.0.0"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.3.5",
			},
		},
		{
			name: "nx_monorepo_yarn_lock_resolves_next_version_from_root_package_json",
			files: map[string]string{
				"nx.json": `{}`,
				"package.json": `{
					"dependencies": {"next": "^16.4.0"}
				}`,
				"yarn.lock": "next@^16.4.0:\n  version \"16.4.3\"\n",
				"apps/web/package.json": `{
					"name": "web",
					"scripts": {"build": "next build"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
			},
		},
		{
			name: "pnpm_v9_monorepo_importers_resolves_locked_next_version",
			files: map[string]string{
				"package.json": `{
					"private": true
				}`,
				"pnpm-lock.yaml": `lockfileVersion: '9.0'
importers:
  .:
    devDependencies:
      turbo:
        specifier: ^2.0.0
        version: 2.0.0
  apps/web:
    dependencies:
      next:
        specifier: ^16.0.0
        version: 16.4.3(react-dom@19.0.0)(react@19.0.0)
`,
				"apps/web/package.json": `{
					"name": "web",
					"scripts": {"build": "next build"},
					"dependencies": {"next": "^16.0.0"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
			},
		},
		{
			name: "npm_monorepo_non_hoisted_resolves_workspace_next_version",
			files: map[string]string{
				"package.json": `{
					"private": true,
					"workspaces": ["apps/*"]
				}`,
				"package-lock.json": `{
					"lockfileVersion": 3,
					"packages": {
						"apps/web/node_modules/next": {"version": "16.4.3"}
					}
				}`,
				"apps/web/package.json": `{
					"name": "web",
					"scripts": {"build": "next build"},
					"dependencies": {"next": "^16.0.0"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
			},
		},
		{
			name: "npm_monorepo_hoisted_resolves_root_next_version",
			files: map[string]string{
				"package.json": `{
					"private": true,
					"workspaces": ["apps/*"]
				}`,
				"package-lock.json": `{
					"lockfileVersion": 3,
					"packages": {
						"node_modules/next": {"version": "16.4.3"}
					}
				}`,
				"apps/web/package.json": `{
					"name": "web",
					"scripts": {"build": "next build"},
					"dependencies": {"next": "^16.0.0"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_BUILDABLE=apps/web"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
			},
		},
		{
			name: "project_npmrc_with_artifact_registry_succeeds",
			files: map[string]string{
				".npmrc": "registry=https://us-central1-npm.pkg.dev/my-project/my-repo/\n//us-central1-npm.pkg.dev/my-project/my-repo/:always-auth=true\n",
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "16.4.3"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.4.3",
			},
		},
		{
			name: "stands_down_when_ar_npmrc_generation_fails",
			files: map[string]string{
				".npmrc/unreadable": "dir",
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "16.4.3"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: failed to generate Artifact Registry credentials",
		},
		{
			name: "stands_down_on_next_below_16_3_5",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": "^15.2.0"}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {"version": "15.2.0"}
					}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: resolved Next.js version \"15.2.0\" is outside the supported range",
		},
		{
			name: "stands_down_on_next_16_3_4",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": "16.3.4"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: resolved Next.js version \"16.3.4\" is outside the supported range",
		},
		{
			name: "stands_down_on_next_17",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": "17.0.0-canary.1"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: resolved Next.js version \"17.0.0-canary.1\" is outside the supported range",
		},
		{
			name: "stands_down_on_next_16_3_5_prerelease",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": "16.3.5-canary.1"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: resolved Next.js version \"16.3.5-canary.1\" is outside the supported range",
		},
		{
			name: "happy_path_next_16_prerelease",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "16.5.0-canary.28"}
				}`,
			},
			envs:           []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantInstallCmd: "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.FrameworkName:    "nextjs",
				bmd.FrameworkVersion: "16.5.0-canary.28",
			},
		},
		{
			name: "stands_down_on_google_function_target",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {"next": "16.4.0"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_FUNCTION_TARGET=myFunc"},
			wantLogSubstring: "Standing down: GOOGLE_FUNCTION_TARGET is set",
		},
		{
			name: "stands_down_on_apphosting_adapter_nextjs",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build"},
					"dependencies": {
						"next": "16.4.0",
						"@apphosting/adapter-nextjs": "14.0.0"
					}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: @apphosting/adapter-nextjs is listed in package.json",
		},
		{
			name: "stands_down_on_custom_server_start_script",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build", "start": "node custom-server.js"},
					"dependencies": {"next": "16.4.0"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: `Standing down: package.json "start" script "node custom-server.js" runs a custom server`,
		},
		{
			name: "stands_down_on_procfile",
			files: map[string]string{
				"Procfile": "web: node server.js",
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "16.4.0"}
				}`,
			},
			envs:             []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			wantLogSubstring: "Standing down: Procfile is present",
		},
		{
			name: "fails_open_when_adapter_install_fails",
			files: map[string]string{
				"package.json": `{
					"scripts": {"build": "next build", "start": "next start"},
					"dependencies": {"next": "16.4.0"}
				}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0`, mockprocess.WithExitCode(1), mockprocess.WithStderr("npm ERR! 404")),
			},
			wantInstallCmd:   "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0",
			wantLogSubstring: "Standing down: failed to install @google-cloud/adapter-nextjs@0.1.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			installCmd := tc.wantInstallCmd
			if installCmd == "" {
				installCmd = "npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0"
			}
			mocks := tc.mocks
			if mocks == nil {
				mocks = []*mockprocess.Mock{
					mockprocess.New(installCmd, mockprocess.WithStdout("installed adapter")),
				}
			}
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithEnvs(tc.envs...),
				bpt.WithExecMocks(mocks...),
			}
			result, err := bpt.RunBuild(t, BuildFn, opts...)
			if err != nil {
				t.Fatalf("RunBuild unexpected error: %v, output: %s", err, result.Output)
			}
			if result.ExitCode != 0 {
				t.Fatalf("ExitCode = %d, want 0; output: %s", result.ExitCode, result.Output)
			}
			wantExecuted := tc.wantInstallCmd != ""
			if got := result.CommandExecuted(installCmd); got != wantExecuted {
				t.Errorf("result.CommandExecuted(%q) = %v, want %v; output: %s", installCmd, got, wantExecuted, result.Output)
			}
			if tc.wantLogSubstring != "" && !strings.Contains(result.Output, tc.wantLogSubstring) {
				t.Errorf("BuildFn() output = %q, want substring %q", result.Output, tc.wantLogSubstring)
			}
			for id, wantVal := range tc.wantMetadata {
				if got := result.MetadataAdded()[id]; got != wantVal {
					t.Errorf("metadata[%s] = %q, want %q", bmd.MetadataIDNames[id], got, wantVal)
				}
			}
		})
	}
}

func TestBuildLayerEnvironmentAndStaleCloudRunRemoval(t *testing.T) {
	appDir := t.TempDir()
	bpDir := t.TempDir()

	eCmd, err := mockprocess.NewExecCmd(
		mockprocess.New(`npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0`, mockprocess.WithStdout("installed")),
	)
	if err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(bpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	// Create stale /workspace/.cloudrun/bundle.yaml to verify BuildFn removes it.
	staleBundle := filepath.Join(appDir, ".cloudrun", "bundle.yaml")
	if err := os.MkdirAll(filepath.Dir(staleBundle), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleBundle, []byte("version: '1'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"dependencies":{"next":"16.4.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOOGLE_WEB_FRAMEWORKS", "true")

	ctx := gcp.NewContext(
		gcp.WithApplicationRoot(appDir),
		gcp.WithBuildpackRoot(bpDir),
		gcp.WithExecCmd(eCmd),
	)
	if err := BuildFn(ctx); err != nil {
		t.Fatalf("BuildFn failed: %v", err)
	}

	// 1. Verify /workspace/.cloudrun was removed.
	if _, err := os.Stat(filepath.Join(appDir, ".cloudrun")); !os.IsNotExist(err) {
		t.Errorf("os.Stat(.cloudrun) err = %v, want os.ErrNotExist", err)
	}

	// 2. Verify layer build environment variables and build-only flags.
	layer := &libcnb.Layer{
		Name: adapterLayerName,
		Path: filepath.Join(bpDir, adapterLayerName),
	}
	if err := gcp.BuildLayer(ctx, layer); err != nil {
		t.Fatal(err)
	}
	if layer.Launch {
		t.Errorf("layer.Launch = true, want false (build-only layer)")
	}
	if layer.Cache {
		t.Errorf("layer.Cache = true, want false (build-only layer)")
	}
	configureAdapterLayerEnv(layer, appDir)

	wantEnvs := map[string]string{
		"NEXT_ADAPTER_PATH.default":              filepath.Join(layer.Path, "node_modules", "@google-cloud", "adapter-nextjs"),
		"NEXT_PRIVATE_OUTPUT_TRACE_ROOT.default": appDir,
		"NEXT_TELEMETRY_DISABLED.default":        "1",
		nodejs.EnvWebAdapterActive + ".override": "true",
	}
	for k, want := range wantEnvs {
		if got := layer.BuildEnvironment[k]; got != want {
			t.Errorf("BuildEnvironment[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestBuildCopiesProjectNPMRCToAdapterLayer(t *testing.T) {
	testCases := []struct {
		name string
		// npmrc is the content of the app's .npmrc; empty means no file.
		npmrc string
	}{
		{
			name:  "copies_project_npmrc",
			npmrc: "registry=https://us-central1-npm.pkg.dev/my-project/my-repo/\n@acme:registry=https://npm.example.com/\n",
		},
		{
			name: "no_project_npmrc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			bpDir := t.TempDir()
			homeDir := t.TempDir()

			// An existing user-level .npmrc makes ar.GenerateNPMConfig skip
			// credential generation, which keeps the test hermetic.
			if err := os.WriteFile(filepath.Join(homeDir, ".npmrc"), nil, 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", homeDir)

			if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"dependencies":{"next":"16.4.0"}}`), 0644); err != nil {
				t.Fatal(err)
			}
			if tc.npmrc != "" {
				if err := os.WriteFile(filepath.Join(appDir, ".npmrc"), []byte(tc.npmrc), 0644); err != nil {
					t.Fatal(err)
				}
			}

			eCmd, err := mockprocess.NewExecCmd(
				mockprocess.New(`npm install --prefix web_nextjs_adapter @google-cloud/adapter-nextjs@0.1.0`, mockprocess.WithStdout("installed")),
			)
			if err != nil {
				t.Fatal(err)
			}

			// The adapter layer path is relative to the working directory when no
			// layers directory is configured.
			origWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(bpDir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(origWd)
			})

			ctx := gcp.NewContext(
				gcp.WithApplicationRoot(appDir),
				gcp.WithBuildpackRoot(bpDir),
				gcp.WithExecCmd(eCmd),
			)
			if err := BuildFn(ctx); err != nil {
				t.Fatalf("BuildFn failed: %v", err)
			}

			layerNPMRC := filepath.Join(bpDir, adapterLayerName, ".npmrc")
			got, err := os.ReadFile(layerNPMRC)
			if tc.npmrc == "" {
				if !os.IsNotExist(err) {
					t.Errorf("os.ReadFile(%q) err = %v, want os.ErrNotExist", layerNPMRC, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("os.ReadFile(%q) failed: %v", layerNPMRC, err)
			}
			if string(got) != tc.npmrc {
				t.Errorf("adapter layer .npmrc = %q, want %q", got, tc.npmrc)
			}
		})
	}
}
