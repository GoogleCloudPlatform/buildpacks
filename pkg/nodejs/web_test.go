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

package nodejs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcpbuildpack "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestWebFrameworksEnabled(t *testing.T) {
	testCases := []struct {
		name           string
		webFrameworks  string
		targetPlatform string
		releaseTrack   string
		envs           map[string]string
		want           bool
	}{
		{
			name: "default_off",
			want: false,
		},
		{
			name:          "true_on_gcp_but_no_track",
			webFrameworks: "true",
			want:          false,
		},
		{
			name:          "true_on_gcp_alpha_track",
			webFrameworks: "true",
			releaseTrack:  "ALPHA",
			want:          true,
		},
		{
			name:          "true_on_gcp_beta_track",
			webFrameworks: "true",
			releaseTrack:  "BETA",
			want:          false,
		},
		{
			name:          "true_on_gcp_ga_track",
			webFrameworks: "true",
			releaseTrack:  "GA",
			want:          false,
		},
		{
			name:          "TRUE_case_insensitive_alpha",
			webFrameworks: "TRUE",
			releaseTrack:  "ALPHA",
			want:          true,
		},
		{
			name:          "1_on_gcp_alpha",
			webFrameworks: "1",
			releaseTrack:  "ALPHA",
			want:          true,
		},
		{
			name:          "false_explicit_alpha",
			webFrameworks: "false",
			releaseTrack:  "ALPHA",
			want:          false,
		},
		{
			name:           "true_alpha_but_on_fah",
			webFrameworks:  "true",
			releaseTrack:   "ALPHA",
			targetPlatform: "fah",
			want:           false,
		},
		{
			name:           "true_alpha_but_on_gcf",
			webFrameworks:  "true",
			releaseTrack:   "ALPHA",
			targetPlatform: "gcf",
			want:           false,
		},
		{
			name:           "true_alpha_but_on_gae",
			webFrameworks:  "true",
			releaseTrack:   "ALPHA",
			targetPlatform: "gae",
			want:           false,
		},
		{
			name:          "true_alpha_but_dev_mode",
			webFrameworks: "true",
			releaseTrack:  "ALPHA",
			envs:          map[string]string{env.DevMode: "true"},
			want:          false,
		},
		{
			name:          "true_alpha_dev_mode_false",
			webFrameworks: "true",
			releaseTrack:  "ALPHA",
			envs:          map[string]string{env.DevMode: "false"},
			want:          true,
		},
		{
			name:          "true_alpha_but_dev_sync",
			webFrameworks: "true",
			releaseTrack:  "ALPHA",
			envs:          map[string]string{env.XGoogleDevSyncActivated: "true", env.DevSync: "true"},
			want:          false,
		},
		{
			name:          "true_alpha_dev_sync_without_activation_flag",
			webFrameworks: "true",
			releaseTrack:  "ALPHA",
			envs:          map[string]string{env.DevSync: "true"},
			want:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvWebFrameworks, tc.webFrameworks)
			t.Setenv("X_GOOGLE_TARGET_PLATFORM", tc.targetPlatform)
			t.Setenv(env.ReleaseTrack, tc.releaseTrack)
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			if got := WebFrameworksEnabled(); got != tc.want {
				t.Errorf("WebFrameworksEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWebAdapterActive(t *testing.T) {
	testCases := []struct {
		name           string
		webFrameworks  string
		adapterActive  string
		targetPlatform string
		releaseTrack   string
		want           bool
	}{
		{
			name: "default_off",
			want: false,
		},
		{
			name:          "adapter_active_but_gate_unset_returns_false",
			adapterActive: "true",
			want:          false,
		},
		{
			name:          "adapter_active_but_gate_false_returns_false",
			webFrameworks: "false",
			adapterActive: "true",
			releaseTrack:  "ALPHA",
			want:          false,
		},
		{
			name:           "adapter_active_and_gate_true_but_gae_returns_false",
			webFrameworks:  "true",
			adapterActive:  "true",
			targetPlatform: "gae",
			releaseTrack:   "ALPHA",
			want:           false,
		},
		{
			name:           "adapter_active_and_gate_true_but_fah_returns_false",
			webFrameworks:  "true",
			adapterActive:  "true",
			targetPlatform: "fah",
			releaseTrack:   "ALPHA",
			want:           false,
		},
		{
			name:          "adapter_active_and_gate_true_on_gcp_but_no_track_returns_false",
			webFrameworks: "true",
			adapterActive: "true",
			want:          false,
		},
		{
			name:          "adapter_active_and_gate_true_on_gcp_alpha_returns_true",
			webFrameworks: "true",
			adapterActive: "true",
			releaseTrack:  "ALPHA",
			want:          true,
		},
		{
			name:          "adapter_active_1_and_gate_1_on_gcp_alpha_returns_true",
			webFrameworks: "1",
			adapterActive: "1",
			releaseTrack:  "ALPHA",
			want:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvWebFrameworks, tc.webFrameworks)
			t.Setenv(EnvWebAdapterActive, tc.adapterActive)
			t.Setenv("X_GOOGLE_TARGET_PLATFORM", tc.targetPlatform)
			t.Setenv(env.ReleaseTrack, tc.releaseTrack)
			if got := WebAdapterActive(); got != tc.want {
				t.Errorf("WebAdapterActive() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecodeYAML(t *testing.T) {
	var got struct {
		Version string `yaml:"version"`
	}
	if err := DecodeYAML([]byte("version: \"1\"\n"), &got); err != nil {
		t.Fatalf("DecodeYAML() unexpected error: %v", err)
	}
	if got.Version != "1" {
		t.Errorf("DecodeYAML() Version = %q, want %q", got.Version, "1")
	}
}

func TestWorkspaceLockfileVersion(t *testing.T) {
	testCases := []struct {
		name      string
		files     map[string]string
		appSubdir string
		pkg       string
		want      string
	}{
		{
			name: "pnpm_v9_importers_monorepo",
			files: map[string]string{
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
			},
			appSubdir: "apps/web",
			pkg:       "next",
			want:      "16.4.3",
		},
		{
			name: "pnpm_v6_importers_dev_dependencies",
			files: map[string]string{
				"pnpm-lock.yaml": `lockfileVersion: '6.0'
importers:
  apps/web:
    devDependencies:
      next:
        version: 16.4.3(react@19.0.0)
`,
			},
			appSubdir: "apps/web",
			pkg:       "next",
			want:      "16.4.3",
		},
		{
			name: "pnpm_importers_direct_string_version",
			files: map[string]string{
				"pnpm-lock.yaml": `importers:
  apps/web:
    dependencies:
      next: 16.4.3(react@19.0.0)
`,
			},
			appSubdir: "apps/web",
			pkg:       "next",
			want:      "16.4.3",
		},
		{
			name: "npm_monorepo_non_hoisted_prefers_workspace_entry",
			files: map[string]string{
				"package-lock.json": `{
					"lockfileVersion": 3,
					"packages": {
						"node_modules/next": {"version": "15.2.0"},
						"apps/web/node_modules/next": {"version": "16.4.3"}
					}
				}`,
			},
			appSubdir: "apps/web",
			pkg:       "next",
			want:      "16.4.3",
		},
		{
			name: "npm_monorepo_hoisted_falls_back_to_root_node_modules",
			files: map[string]string{
				"package-lock.json": `{
					"lockfileVersion": 3,
					"packages": {
						"node_modules/next": {"version": "16.4.3"}
					}
				}`,
			},
			appSubdir: "apps/web",
			pkg:       "next",
			want:      "16.4.3",
		},
		{
			name: "single_app_returns_empty",
			files: map[string]string{
				"package-lock.json": `{
					"lockfileVersion": 3,
					"packages": {
						"node_modules/next": {"version": "16.4.3"}
					}
				}`,
			},
			appSubdir: "",
			pkg:       "next",
			want:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := t.TempDir()
			for rel, content := range tc.files {
				full := filepath.Join(rootDir, rel)
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatalf("os.MkdirAll(%q) failed: %v", filepath.Dir(full), err)
				}
				if err := os.WriteFile(full, []byte(content), 0644); err != nil {
					t.Fatalf("os.WriteFile(%q) failed: %v", full, err)
				}
			}
			appDir := rootDir
			if tc.appSubdir != "" {
				appDir = filepath.Join(rootDir, tc.appSubdir)
			}
			got, err := WorkspaceLockfileVersion(rootDir, appDir, tc.pkg)
			if err != nil {
				t.Fatalf("WorkspaceLockfileVersion(%q, %q, %q) unexpected error: %v", rootDir, appDir, tc.pkg, err)
			}
			if got != tc.want {
				t.Errorf("WorkspaceLockfileVersion(%q, %q, %q) = %q, want %q", rootDir, appDir, tc.pkg, got, tc.want)
			}
		})
	}
}

func TestBunLockVersion_MissingFile(t *testing.T) {
	dir := t.TempDir()
	ver, err := BunLockVersion(dir, "next")
	if err != nil {
		t.Fatalf("BunLockVersion(%q, %q) unexpected error: %v", dir, "next", err)
	}
	if ver != "" {
		t.Errorf("BunLockVersion(%q, %q) = %q, want empty", dir, "next", ver)
	}
}

func TestBunLockVersion(t *testing.T) {
	dir := t.TempDir()
	bunLockContent := `{
  // Bun lockfile v1 with comments and trailing commas
  "lockfileVersion": 1,
  "workspaces": {
    "": {
      "name": "my-bun-app",
      "dependencies": {
        "next": "^16.4.3",
      },
    },
  },
  /* block comment */
  "packages": {
    "next": ["next@16.4.3", "", { "dependencies": { "react": "^19.0.0", }, }, "sha512-abc"],
    "@next/routing": ["@next/routing@16.4.3", "", {}, "sha512-def",],
  },
}
`
	if err := os.WriteFile(filepath.Join(dir, "bun.lock"), []byte(bunLockContent), 0644); err != nil {
		t.Fatalf("os.WriteFile(bun.lock) failed: %v", err)
	}

	testCases := []struct {
		name string
		pkg  string
		want string
	}{
		{
			name: "unscoped_package",
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "scoped_package",
			pkg:  "@next/routing",
			want: "16.4.3",
		},
		{
			name: "nonexistent_package",
			pkg:  "nonexistent",
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BunLockVersion(dir, tc.pkg)
			if err != nil {
				t.Fatalf("BunLockVersion(%q, %q) unexpected error: %v", dir, tc.pkg, err)
			}
			if got != tc.want {
				t.Errorf("BunLockVersion(%q, %q) = %q, want %q", dir, tc.pkg, got, tc.want)
			}
		})
	}
}

func TestWebStandDownReason(t *testing.T) {
	testCases := []struct {
		name      string
		files     map[string]string
		envs      map[string]string
		appSubdir string
		wantAllow bool
	}{
		{
			name: "plain_next_start",
			files: map[string]string{
				"package.json": `{"scripts":{"build":"next build","start":"next start"}}`,
			},
			wantAllow: true,
		},
		{
			name: "next_start_with_port_and_hostname_flags",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start -p 8080 -H 0.0.0.0"}}`,
			},
			wantAllow: true,
		},
		{
			name: "next_start_with_long_equals_flags",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start --port=3000 --hostname=0.0.0.0"}}`,
			},
			wantAllow: true,
		},
		{
			name: "node_standalone_server_js",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"node .next/standalone/server.js"}}`,
			},
			wantAllow: true,
		},
		{
			name: "no_start_script_and_no_server_js_or_main",
			files: map[string]string{
				"package.json": `{"scripts":{"build":"next build"}}`,
			},
			wantAllow: true,
		},
		{
			name: "custom_server_node_server_js_in_start",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"node server.js"}}`,
			},
			wantAllow: false,
		},
		{
			name: "unsupported_next_start_flag",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start --experimental-https"}}`,
			},
			wantAllow: false,
		},
		{
			name: "env_prefix_on_next_start",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"NODE_OPTIONS=--inspect next start"}}`,
			},
			wantAllow: false,
		},
		{
			name: "cross_env_prefix_on_next_start",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"cross-env PORT=3000 next start"}}`,
			},
			wantAllow: false,
		},
		{
			name: "no_start_script_but_server_js_present",
			files: map[string]string{
				"package.json": `{"scripts":{"build":"next build"}}`,
				"server.js":    `console.log("custom");`,
			},
			wantAllow: false,
		},
		{
			name: "no_start_script_but_main_present",
			files: map[string]string{
				"package.json": `{"main":"custom.js","scripts":{"build":"next build"}}`,
			},
			wantAllow: false,
		},
		{
			name: "procfile_present",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
				"Procfile":     "web: node custom.js\n",
			},
			wantAllow: false,
		},
		{
			name: "google_entrypoint_set",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Entrypoint: "node custom.js",
			},
			wantAllow: false,
		},
		{
			name: "monorepo_with_app_scoped_turbo_start",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"turbo run start --filter=web"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: true,
		},
		{
			name: "monorepo_with_app_scoped_nx_start",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"nx run web:start"}}`,
				"apps/web/project.json": `{"name":"web"}`,
				"apps/web/package.json": `{"scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: true,
		},
		{
			name: "monorepo_with_app_scoped_npm_workspace_start",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"npm run start -w apps/web"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: true,
		},
		{
			name: "monorepo_with_app_scoped_pnpm_filter_start",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"pnpm --filter web start"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: true,
		},
		{
			name: "monorepo_with_app_scoped_yarn_workspace_start",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"yarn workspace web start"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: true,
		},
		{
			name: "monorepo_with_root_custom_start_stands_down",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"node root-server.js"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
			},
			envs: map[string]string{
				env.Buildable: "apps/web",
			},
			appSubdir: "apps/web",
			wantAllow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := t.TempDir()
			for rel, content := range tc.files {
				full := filepath.Join(rootDir, rel)
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}
				if err := os.WriteFile(full, []byte(content), 0644); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			}
			t.Setenv(env.Entrypoint, "")
			t.Setenv(env.Buildable, "")
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			appDir := rootDir
			if tc.appSubdir != "" {
				appDir = filepath.Join(rootDir, tc.appSubdir)
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(rootDir))
			reason, err := WebStandDownReason(ctx, appDir)
			if err != nil {
				t.Fatalf("WebStandDownReason() unexpected error: %v", err)
			}
			gotAllow := reason == ""
			if gotAllow != tc.wantAllow {
				t.Errorf("WebStandDownReason() reason = %q (allow=%v), want allow=%v", reason, gotAllow, tc.wantAllow)
			}
		})
	}
}
