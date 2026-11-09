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
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
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
			name: "opts_out_when_GOOGLE_WEB_FRAMEWORKS_not_set",
			files: map[string]string{
				"next.config.mjs": "export default {};",
				"package.json":    `{"dependencies":{"next":"15.2.0"}}`,
			},
			want: 100,
		},
		{
			name: "opts_out_when_target_platform_is_fah",
			files: map[string]string{
				"next.config.mjs": "export default {};",
			},
			envs: []string{
				"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA",
				"X_GOOGLE_TARGET_PLATFORM=fah",
			},
			want: 100,
		},
		{
			name: "opts_out_when_not_a_nextjs_app",
			files: map[string]string{
				"package.json": `{"dependencies":{"express":"4.19.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 100,
		},
		{
			name: "opts_in_with_next_config_ts",
			files: map[string]string{
				"next.config.ts": "export default {};",
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_with_next_dependency_in_package_json",
			files: map[string]string{
				"package.json": `{"dependencies":{"next":"15.2.0"}}`,
			},
			envs: []string{"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "opts_in_for_monorepo_app_via_GOOGLE_BUILDABLE",
			files: map[string]string{
				"apps/web/next.config.mjs": "export default {};",
			},
			envs: []string{
				"GOOGLE_WEB_FRAMEWORKS=true", "X_GOOGLE_RELEASE_TRACK=ALPHA",
				"GOOGLE_BUILDABLE=apps/web",
			},
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
		name         string
		files        map[string]string
		envs         map[string]string
		initialCmd   []string
		wantCmd      []string
		wantLabels   []libcnb.Label
		wantExist    []string
		wantNotExist []string
	}{
		{
			name: "no_manifest_is_noop",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist:  []string{"package.json"},
		},
		{
			name: "default_report_only_mode_deletes_standalone_and_sets_project_run_command",
			files: map[string]string{
				"package.json":                                      `{"scripts":{"start":"next start"}}`,
				"src/app/page.tsx":                                  "export default function Page() {}",
				"data/Catalog.json":                                 `{"items":[]}`,
				"public/logo.png":                                   "logo",
				".cloudrun/bundle.yaml":                             validBundleYAML,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/required-server-files.json":                  `{"version":1}`,
				".next/static/chunks/app.js":                        "chunk",
				".next/cache/webpack/0.pack":                        "cache",
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js", "--project"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "project"},
			},
			wantExist: []string{
				".cloudrun/bundle.yaml",
				".cloudrun/serve.js",
				"src/app/page.tsx",
				"data/Catalog.json",
				".next/static/chunks/app.js",
			},
			wantNotExist: []string{
				".next/standalone",
			},
		},
		{
			name: "prune_true_copies_assets_prunes_workspace_and_sets_standalone_command",
			files: map[string]string{
				"package.json":                                      `{"scripts":{"start":"next start"}}`,
				"src/app/page.tsx":                                  "export default function Page() {}",
				"public/logo.png":                                   "logo",
				".cloudrun/bundle.yaml":                             validBundleYAML,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/static/chunks/app.js":                        "chunk",
				".next/cache/webpack/0.pack":                        "cache",
				".next/standalone/package.json":                     `{"name":"app"}`,
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "standalone"},
			},
			wantExist: []string{
				".cloudrun/bundle.yaml",
				".cloudrun/serve.js",
				".next/standalone/package.json",
				".next/standalone/server.js",
				".next/standalone/node_modules/next/package.json",
				".next/standalone/public/logo.png",
				".next/standalone/.next/static/chunks/app.js",
			},
			wantNotExist: []string{
				"src",
				"public",
				".next/cache",
				".next/static",
			},
		},
		{
			name: "prune_false_deletes_standalone_and_sets_project_run_command",
			files: map[string]string{
				"package.json":                                      `{"scripts":{"start":"next start"}}`,
				"src/app/page.tsx":                                  "export default function Page() {}",
				".cloudrun/bundle.yaml":                             validBundleYAML,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/required-server-files.json":                  `{"version":1}`,
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "false",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js", "--project"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "project"},
			},
			wantExist: []string{
				"src/app/page.tsx",
				".cloudrun/serve.js",
			},
			wantNotExist: []string{
				".next/standalone",
			},
		},
		{
			name: "report_only_missing_project_required_server_files_keeps_initial_start_command",
			files: map[string]string{
				"package.json":                                      `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml":                             validBundleYAML,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
				// Note: .next/required-server-files.json is missing!
			},
			initialCmd:   []string{"npm", "run", "start"},
			wantCmd:      []string{"npm", "run", "start"},
			wantNotExist: []string{".next/standalone"},
		},
		{
			name: "missing_next_routing_json_when_using_serve_js_keeps_initial_start_command",
			files: map[string]string{
				"package.json":                                      `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml":                             validBundleYAML,
				".cloudrun/serve.js":                                "console.log('serve');",
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
		},
		{
			name: "unknown_version_warns_and_keeps_initial_start_command",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v1"
runConfig:
  runCommand: "node .next/standalone/server.js"
metadata:
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include: [".next/standalone"]
`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
		},
		{
			name: "export_mode_warns_and_stands_down",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
metadata:
  outputMode: "export"
  projectDir: "."
  distDir: "out"
`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
		},
		{
			name: "custom_start_script_stands_down_and_keeps_user_start",
			files: map[string]string{
				"package.json":                                    `{"scripts":{"start":"node custom-server.js"}}`,
				".cloudrun/bundle.yaml":                           validBundleYAML,
				".cloudrun/serve.js":                              "console.log('serve');",
				".cloudrun/next-routing.json":                     `{"version":1}`,
				".next/standalone/server.js":                      "console.log('standalone');",
				".next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist:  []string{".next/standalone/server.js"},
		},
		{
			name: "failed_standalone_check_with_valid_fallback_files_uses_project_run_command",
			files: map[string]string{
				"package.json":                     `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml":            validBundleYAML,
				".cloudrun/serve.js":               "console.log('serve');",
				".cloudrun/next-routing.json":      `{"version":1}`,
				".next/required-server-files.json": `{"version":1}`,
				// Note: .next/standalone/server.js is missing!
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js", "--project"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "project"},
			},
		},
		{
			name: "failed_standalone_check_without_fallback_files_keeps_initial_start_command",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": validBundleYAML,
				".cloudrun/serve.js":    "console.log('serve');",
				// Both .next/standalone/server.js and .next/required-server-files.json are missing!
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
		},
		{
			name: "corrupt_next_routing_json_keeps_initial_start_command",
			files: map[string]string{
				"package.json":                                    `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml":                           validBundleYAML,
				".cloudrun/serve.js":                              "console.log('serve');",
				".cloudrun/next-routing.json":                     `{not-valid-json`,
				".next/standalone/server.js":                      "console.log('standalone');",
				".next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
		},
		{
			name: "report_only_without_project_run_command_keeps_initial_start_command_and_no_labels",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  outputMode: "standalone"
  projectDir: "."
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
`,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			initialCmd:   []string{"npm", "run", "start"},
			wantCmd:      []string{"npm", "run", "start"},
			wantNotExist: []string{".next/standalone"},
		},
		{
			name: "report_only_without_project_run_command_and_user_standalone_start_preserves_standalone",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"node .next/standalone/server.js"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  outputMode: "standalone"
  projectDir: "."
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
`,
				".cloudrun/serve.js":                                "console.log('serve');",
				".cloudrun/next-routing.json":                       `{"version":1}`,
				".next/standalone/server.js":                        "console.log('standalone');",
				".next/standalone/.next/required-server-files.json": `{"version":1}`,
				".next/standalone/node_modules/next/package.json":   `{"name":"next"}`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist:  []string{".next/standalone/server.js"},
		},
		{
			name: "failed_standalone_check_with_unincluded_distDir_does_not_delete_user_source",
			files: map[string]string{
				"package.json":        `{"scripts":{"start":"next start"}}`,
				"src/standalone/a.ts": "export const a = 1;",
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "."
  distDir: "src"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
`,
				".cloudrun/serve.js":          "console.log('serve');",
				".cloudrun/next-routing.json": `{"version":1}`,
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist:  []string{"src/standalone/a.ts"},
		},
		{
			name: "non_serve_js_run_command_sets_process_without_serve_js_labels",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .next/standalone/server.js"
metadata:
  outputMode: "standalone"
  projectDir: "."
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
`,
				".next/standalone/package.json":                   `{"name":"app"}`,
				".next/standalone/server.js":                      "console.log('standalone');",
				".next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".next/standalone/server.js"},
		},
		{
			name: "monorepo_prune_true_with_app_scoped_root_start_succeeds",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"turbo run start --filter=web"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "apps/web"
  distDir: "apps/web/.next"
  serverDir: "apps/web/.next/standalone/apps/web"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "apps/web/.next/standalone"
      - ".cloudrun"
    copy:
      - from: "apps/web/public"
        to: "apps/web/.next/standalone/apps/web/public"
`,
				".cloudrun/serve.js":                                                  "console.log('serve');",
				".cloudrun/next-routing.json":                                         `{"version":1}`,
				"apps/web/public/favicon.ico":                                         "icon",
				"apps/web/.next/standalone/apps/web/package.json":                     `{"name":"web"}`,
				"apps/web/.next/standalone/apps/web/server.js":                        "console.log('standalone');",
				"apps/web/.next/standalone/apps/web/.next/required-server-files.json": `{"version":1}`,
				"apps/web/.next/standalone/node_modules/next/package.json":            `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
				"GOOGLE_BUILDABLE": "apps/web",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "standalone"},
			},
			wantExist: []string{
				"apps/web/.next/standalone/apps/web/package.json",
				"apps/web/.next/standalone/apps/web/server.js",
				"apps/web/.next/standalone/apps/web/public/favicon.ico",
			},
			wantNotExist: []string{
				"apps/web/public",
			},
		},
		{
			name: "monorepo_report_only_with_boot_package_json_present_succeeds",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"turbo run start --filter=web"}}`,
				"apps/web/package.json": `{"name":"web","scripts":{"start":"next start"}}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "apps/web"
  distDir: "apps/web/.next"
  serverDir: "apps/web/.next/standalone/apps/web"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "apps/web/.next/standalone"
      - ".cloudrun"
`,
				".cloudrun/serve.js":                                                  "console.log('serve');",
				".cloudrun/next-routing.json":                                         `{"version":1}`,
				"apps/web/.next/required-server-files.json":                           `{"version":1}`,
				"apps/web/.next/standalone/apps/web/server.js":                        "console.log('standalone');",
				"apps/web/.next/standalone/apps/web/.next/required-server-files.json": `{"version":1}`,
				"apps/web/.next/standalone/node_modules/next/package.json":            `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_BUILDABLE": "apps/web",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"node", ".cloudrun/serve.js", "--project"},
			wantLabels: []libcnb.Label{
				{Key: "google.web-cache-guard", Value: "true"},
				{Key: "google.web-serve-mode", Value: "project"},
			},
			wantExist: []string{
				"apps/web/package.json",
			},
			wantNotExist: []string{
				"apps/web/.next/standalone",
			},
		},
		{
			name: "nx_monorepo_report_only_missing_project_package_json_warns_and_preserves_standalone",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"nx run web:serve"}}`,
				"nx.json":               `{}`,
				"apps/web/project.json": `{"name":"web"}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "apps/web"
  distDir: "apps/web/.next"
  serverDir: "apps/web/.next/standalone/apps/web"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "apps/web/.next/standalone"
      - ".cloudrun"
`,
				".cloudrun/serve.js":                                                  "console.log('serve');",
				".cloudrun/next-routing.json":                                         `{"version":1}`,
				"apps/web/.next/required-server-files.json":                           `{"version":1}`,
				"apps/web/.next/standalone/apps/web/server.js":                        "console.log('standalone');",
				"apps/web/.next/standalone/apps/web/.next/required-server-files.json": `{"version":1}`,
				"apps/web/.next/standalone/node_modules/next/package.json":            `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_BUILDABLE": "apps/web",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist: []string{
				"apps/web/.next/standalone/apps/web/server.js",
			},
		},
		{
			name: "nx_monorepo_prune_false_missing_project_package_json_warns_and_preserves_standalone",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"nx run web:serve"}}`,
				"nx.json":               `{}`,
				"apps/web/project.json": `{"name":"web"}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "apps/web"
  distDir: "apps/web/.next"
  serverDir: "apps/web/.next/standalone/apps/web"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "apps/web/.next/standalone"
      - ".cloudrun"
`,
				".cloudrun/serve.js":                                                  "console.log('serve');",
				".cloudrun/next-routing.json":                                         `{"version":1}`,
				"apps/web/.next/required-server-files.json":                           `{"version":1}`,
				"apps/web/.next/standalone/apps/web/server.js":                        "console.log('standalone');",
				"apps/web/.next/standalone/apps/web/.next/required-server-files.json": `{"version":1}`,
				"apps/web/.next/standalone/node_modules/next/package.json":            `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "false",
				"GOOGLE_BUILDABLE": "apps/web",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist: []string{
				"apps/web/.next/standalone/apps/web/server.js",
			},
		},
		{
			name: "nx_monorepo_prune_true_missing_server_package_json_warns_and_skips_pruning",
			files: map[string]string{
				"package.json":          `{"scripts":{"start":"nx run web:serve"}}`,
				"nx.json":               `{}`,
				"apps/web/project.json": `{"name":"web"}`,
				".cloudrun/bundle.yaml": `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  outputMode: "standalone"
  projectDir: "apps/web"
  distDir: "apps/web/.next"
  serverDir: "apps/web/.next/standalone/apps/web"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "apps/web/.next/standalone"
      - ".cloudrun"
    copy:
      - from: "apps/web/public"
        to: "apps/web/.next/standalone/apps/web/public"
`,
				".cloudrun/serve.js":                                                  "console.log('serve');",
				".cloudrun/next-routing.json":                                         `{"version":1}`,
				"apps/web/public/favicon.ico":                                         "icon",
				"apps/web/.next/required-server-files.json":                           `{"version":1}`,
				"apps/web/.next/standalone/apps/web/server.js":                        "console.log('standalone');",
				"apps/web/.next/standalone/apps/web/.next/required-server-files.json": `{"version":1}`,
				"apps/web/.next/standalone/node_modules/next/package.json":            `{"name":"next"}`,
			},
			envs: map[string]string{
				"GOOGLE_WEB_PRUNE": "true",
				"GOOGLE_BUILDABLE": "apps/web",
			},
			initialCmd: []string{"npm", "run", "start"},
			wantCmd:    []string{"npm", "run", "start"},
			wantExist: []string{
				"apps/web/public/favicon.ico",
				"apps/web/.next/standalone/apps/web/server.js",
			},
			wantNotExist: []string{
				"apps/web/.next/standalone/apps/web/public",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			setupWorkspaceFiles(t, ws, tc.files)

			t.Setenv("GOOGLE_WEB_FRAMEWORKS", "true")
			t.Setenv("GOOGLE_WEB_PRUNE", "")
			t.Setenv("GOOGLE_ENTRYPOINT", "")
			t.Setenv("GOOGLE_BUILDABLE", "")
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(ws))
			if len(tc.initialCmd) > 0 {
				ctx.AddWebProcess(tc.initialCmd)
			}

			if err := BuildFn(ctx); err != nil {
				t.Fatalf("BuildFn() returned error %v, want nil (never fail build)", err)
			}

			if len(ctx.Processes()) != 1 {
				t.Fatalf("BuildFn() len(ctx.Processes()) = %d, want 1", len(ctx.Processes()))
			}
			gotCmd := ctx.Processes()[0].Command
			if diff := cmp.Diff(tc.wantCmd, gotCmd); diff != "" {
				t.Errorf("BuildFn() web process command diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantLabels, ctx.Labels()); diff != "" {
				t.Errorf("BuildFn() labels diff (-want +got):\n%s", diff)
			}

			for _, rel := range tc.wantExist {
				if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
					t.Errorf("os.Stat(%q) after BuildFn() failed: %v, want file to exist", rel, err)
				}
			}
			for _, rel := range tc.wantNotExist {
				if _, err := os.Lstat(filepath.Join(ws, filepath.FromSlash(rel))); !os.IsNotExist(err) {
					t.Errorf("os.Lstat(%q) after BuildFn() = %v, want os.IsNotExist", rel, err)
				}
			}
		})
	}
}

const validBundleYAML = `version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  adapterPackageName: "@apphosting/adapter-nextjs"
  adapterVersion: "15.10.1"
  framework: "nextjs"
  frameworkVersion: "15.2.0"
  outputMode: "standalone"
  projectDir: "."
  distDir: ".next"
  serverDir: ".next/standalone"
  runtimeManifest: ".cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
      - ".cloudrun"
    copy:
      - from: "public"
        to: ".next/standalone/public"
      - from: ".next/static"
        to: ".next/standalone/.next/static"
`

func setupWorkspaceFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatalf("MkdirAll(%q) failed: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", abs, err)
		}
	}
}
