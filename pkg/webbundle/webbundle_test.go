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

package webbundle

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCleanAndValidatePath(t *testing.T) {
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", outsideFile, err)
	}

	ws := t.TempDir()
	writeTestFiles(t, ws, map[string]string{
		"apps/web/server.js": "console.log('ok');",
	})
	if err := os.Symlink(outsideFile, filepath.Join(ws, "escape_file")); err != nil {
		t.Fatalf("Symlink escape_file failed: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(ws, "escape_dir")); err != nil {
		t.Fatalf("Symlink escape_dir failed: %v", err)
	}
	if err := os.Symlink("../../../nonexistent_outside", filepath.Join(ws, "broken_escape")); err != nil {
		t.Fatalf("Symlink broken_escape failed: %v", err)
	}
	if err := os.Symlink("apps/web", filepath.Join(ws, "valid_link")); err != nil {
		t.Fatalf("Symlink valid_link failed: %v", err)
	}

	tests := []struct {
		name     string
		rawPath  string
		allowDot bool
		want     string
		wantErr  bool
	}{
		{
			name:     "simple_relative_path",
			rawPath:  ".next/standalone",
			allowDot: false,
			want:     ".next/standalone",
		},
		{
			name:     "leading_dot_slash_cleaned",
			rawPath:  "./.next/standalone",
			allowDot: false,
			want:     ".next/standalone",
		},
		{
			name:     "trailing_slash_cleaned",
			rawPath:  ".next/standalone/",
			allowDot: false,
			want:     ".next/standalone",
		},
		{
			name:     "dot_allowed",
			rawPath:  ".",
			allowDot: true,
			want:     ".",
		},
		{
			name:     "dot_slash_allowed_when_allowDot_true",
			rawPath:  "./",
			allowDot: true,
			want:     ".",
		},
		{
			name:     "dot_disallowed",
			rawPath:  ".",
			allowDot: false,
			wantErr:  true,
		},
		{
			name:     "empty_path_rejected",
			rawPath:  "",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "absolute_path_rejected",
			rawPath:  "/workspace/.next/standalone",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "dot_dot_prefix_rejected",
			rawPath:  "../outside",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "dot_dot_middle_rejected",
			rawPath:  "apps/../../outside",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "dot_dot_in_bounds_still_rejected",
			rawPath:  "apps/web/../web/server.js",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "backslash_rejected",
			rawPath:  `apps\web\server.js`,
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "symlink_escaping_workspace_file_rejected",
			rawPath:  "escape_file",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "symlink_escaping_workspace_dir_subpath_rejected",
			rawPath:  "escape_dir/sub/file.js",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "broken_symlink_escaping_workspace_rejected",
			rawPath:  "broken_escape",
			allowDot: true,
			wantErr:  true,
		},
		{
			name:     "valid_in_workspace_symlink_allowed",
			rawPath:  "valid_link/server.js",
			allowDot: false,
			want:     "valid_link/server.js",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CleanAndValidatePath(ws, tc.rawPath, tc.allowDot)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CleanAndValidatePath(%q, allowDot=%v) error = %v, wantErr %v", tc.rawPath, tc.allowDot, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("CleanAndValidatePath(%q, allowDot=%v) = %q, want %q", tc.rawPath, tc.allowDot, got, tc.want)
			}
		})
	}
}

func TestParseAndValidate(t *testing.T) {
	ws := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.Symlink(outsideDir, filepath.Join(ws, "outside_link")); err != nil {
		t.Fatalf("Symlink outside_link failed: %v", err)
	}

	tests := []struct {
		name      string
		yaml      string
		want      *BundleSchema
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "valid_v2_root_app",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  adapterPackageName: "@apphosting/adapter-nextjs"
  adapterVersion: "15.10.1"
  framework: "nextjs"
  frameworkVersion: "15.2.0"
  buildId: "build-123"
  outputMode: "standalone"
  projectDir: "./"
  distDir: "./.next/"
  serverDir: "./.next/standalone/"
  runtimeManifest: "./.cloudrun/next-routing.json"
outputFiles:
  serverApp:
    include:
      - "./.next/standalone/"
    copy:
      - from: "./public/"
        to: "./.next/standalone/public/"
      - from: "./.next/static/"
        to: "./.next/standalone/.next/static/"
`,
			want: &BundleSchema{
				Version: "v2",
				RunConfig: RunConfig{
					RunCommand:        "node .cloudrun/serve.js",
					ProjectRunCommand: "node .cloudrun/serve.js --project",
				},
				Metadata: Metadata{
					AdapterPackageName: "@apphosting/adapter-nextjs",
					AdapterVersion:     "15.10.1",
					Framework:          "nextjs",
					FrameworkVersion:   "15.2.0",
					BuildID:            "build-123",
					OutputMode:         "standalone",
					ProjectDir:         ".",
					DistDir:            ".next",
					ServerDir:          ".next/standalone",
					RuntimeManifest:    ".cloudrun/next-routing.json",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{".next/standalone", ".cloudrun"},
						Copy: []CopySpec{
							{From: "public", To: ".next/standalone/public"},
							{From: ".next/static", To: ".next/standalone/.next/static"},
						},
					},
				},
			},
		},
		{
			name: "valid_v2_monorepo_app",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
  projectRunCommand: "node .cloudrun/serve.js --project"
metadata:
  adapterPackageName: "@apphosting/adapter-nextjs"
  adapterVersion: "15.10.1"
  framework: "nextjs"
  frameworkVersion: "15.2.0"
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
      - from: "apps/web/.next/static"
        to: "apps/web/.next/standalone/apps/web/.next/static"
`,
			want: &BundleSchema{
				Version: "v2",
				RunConfig: RunConfig{
					RunCommand:        "node .cloudrun/serve.js",
					ProjectRunCommand: "node .cloudrun/serve.js --project",
				},
				Metadata: Metadata{
					AdapterPackageName: "@apphosting/adapter-nextjs",
					AdapterVersion:     "15.10.1",
					Framework:          "nextjs",
					FrameworkVersion:   "15.2.0",
					OutputMode:         "standalone",
					ProjectDir:         "apps/web",
					DistDir:            "apps/web/.next",
					ServerDir:          "apps/web/.next/standalone/apps/web",
					RuntimeManifest:    ".cloudrun/next-routing.json",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{"apps/web/.next/standalone", ".cloudrun"},
						Copy: []CopySpec{
							{From: "apps/web/public", To: "apps/web/.next/standalone/apps/web/public"},
							{From: "apps/web/.next/static", To: "apps/web/.next/standalone/apps/web/.next/static"},
						},
					},
				},
			},
		},
		{
			name: "unknown_version_v1_rejected",
			yaml: `
version: "v1"
runConfig:
  runCommand: "node server.js"
metadata:
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include: [".next/standalone"]
`,
			wantErr: true,
		},
		{
			name: "missing_version_rejected",
			yaml: `
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include: [".next/standalone"]
`,
			wantErr: true,
		},
		{
			name: "export_mode_returns_ErrExportMode",
			yaml: `
version: "v2"
runConfig:
  runCommand: ""
metadata:
  outputMode: "export"
  projectDir: "."
  distDir: "out"
`,
			wantErr:   true,
			wantErrIs: ErrExportMode,
		},
		{
			name: "empty_include_returns_ErrEmptyInclude",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include: []
`,
			wantErr:   true,
			wantErrIs: ErrEmptyInclude,
		},
		{
			name: "cache_dir_in_include_rejected",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
      - ".next/cache/webpack"
`,
			wantErr: true,
		},
		{
			name: "serverDir_outside_include_rejected",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  distDir: ".next"
  serverDir: "other/dir"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
`,
			wantErr: true,
		},
		{
			name: "copy_to_outside_include_rejected",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
    copy:
      - from: "public"
        to: "outside_standalone/public"
`,
			wantErr: true,
		},
		{
			name: "symlink_escape_in_include_rejected",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - ".next/standalone"
      - "outside_link/sub"
`,
			wantErr: true,
		},
		{
			name: "overlapping_copy_from_and_to_rejected",
			yaml: `
version: "v2"
runConfig:
  runCommand: "node .cloudrun/serve.js"
metadata:
  distDir: ".next"
  serverDir: ".next/standalone"
outputFiles:
  serverApp:
    include:
      - "."
    copy:
      - from: "public"
        to: "public/sub"
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAndValidate(ws, []byte(tc.yaml))
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseAndValidate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("ParseAndValidate() error = %v, want errors.Is(_, %v)", err, tc.wantErrIs)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("ParseAndValidate() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCheckStandalone(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		schema  *BundleSchema
		wantErr bool
	}{
		{
			name: "valid_root_standalone",
			files: map[string]string{
				".next/standalone/server.js":                      "console.log('server');",
				".next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			schema: &BundleSchema{
				Metadata: Metadata{
					ProjectDir: ".",
					DistDir:    ".next",
					ServerDir:  ".next/standalone",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{".next/standalone", ".cloudrun"},
					},
				},
			},
		},
		{
			name: "valid_monorepo_standalone_next_in_standalone_root",
			files: map[string]string{
				"apps/web/.next/standalone/apps/web/server.js":             "console.log('server');",
				"apps/web/.next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			schema: &BundleSchema{
				Metadata: Metadata{
					ProjectDir: "apps/web",
					DistDir:    "apps/web/.next",
					ServerDir:  "apps/web/.next/standalone/apps/web",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{"apps/web/.next/standalone", ".cloudrun"},
					},
				},
			},
		},
		{
			name: "missing_server_js_fails",
			files: map[string]string{
				".next/standalone/node_modules/next/package.json": `{"name":"next"}`,
			},
			schema: &BundleSchema{
				Metadata: Metadata{
					ProjectDir: ".",
					DistDir:    ".next",
					ServerDir:  ".next/standalone",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{".next/standalone", ".cloudrun"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "next_package_only_outside_standalone_root_fails",
			files: map[string]string{
				"apps/web/.next/standalone/apps/web/server.js": "console.log('server');",
				"node_modules/next/package.json":               `{"name":"next"}`,
			},
			schema: &BundleSchema{
				Metadata: Metadata{
					ProjectDir: "apps/web",
					DistDir:    "apps/web/.next",
					ServerDir:  "apps/web/.next/standalone/apps/web",
				},
				OutputFiles: OutputFiles{
					ServerApp: ServerApp{
						Include: []string{"apps/web/.next/standalone", ".cloudrun"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			writeTestFiles(t, ws, tc.files)
			err := CheckStandalone(ws, tc.schema)
			if (err != nil) != tc.wantErr {
				t.Errorf("CheckStandalone() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMergeCopy(t *testing.T) {
	ws := t.TempDir()
	writeTestFiles(t, ws, map[string]string{
		"shared-assets/brand.svg":                           "<svg>brand</svg>",
		"public/favicon.ico":                                "new-favicon",
		"public/images/logo.png":                            "logo-bytes",
		".next/static/chunks/main.js":                       "main-chunk",
		".next/standalone/public/existing.txt":              "keep-me",
		".next/standalone/public/favicon.ico":               "old-favicon",
		".next/standalone/.next/server/pages-manifest.json": `{}`,
	})
	if err := os.Symlink("../shared-assets/brand.svg", filepath.Join(ws, "public/brand.svg")); err != nil {
		t.Fatalf("Symlink public/brand.svg failed: %v", err)
	}

	specs := []CopySpec{
		{From: "public", To: ".next/standalone/public"},
		{From: ".next/static", To: ".next/standalone/.next/static"},
		{From: "missing_optional_dir", To: ".next/standalone/missing_optional_dir"},
	}

	if err := MergeCopy(ws, specs); err != nil {
		t.Fatalf("MergeCopy() unexpected error: %v", err)
	}

	wantContents := map[string]string{
		".next/standalone/public/existing.txt":              "keep-me",
		".next/standalone/public/favicon.ico":               "new-favicon",
		".next/standalone/public/images/logo.png":           "logo-bytes",
		".next/standalone/public/brand.svg":                 "<svg>brand</svg>",
		".next/standalone/.next/static/chunks/main.js":      "main-chunk",
		".next/standalone/.next/server/pages-manifest.json": `{}`,
	}
	for rel, want := range wantContents {
		gotBytes, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("os.ReadFile(%q) after MergeCopy() failed: %v", rel, err)
			continue
		}
		if string(gotBytes) != want {
			t.Errorf("os.ReadFile(%q) after MergeCopy() = %q, want %q", rel, string(gotBytes), want)
		}
	}

	if err := MergeCopy(ws, []CopySpec{{From: "public", To: "public"}}); err == nil {
		t.Errorf("MergeCopy() with overlapping From/To = nil, want non-nil error")
	}
}

func TestCheckBootPackageJSON(t *testing.T) {
	schema := &BundleSchema{
		Metadata: Metadata{
			ProjectDir: "apps/web",
			DistDir:    "apps/web/.next",
			ServerDir:  "apps/web/.next/standalone/apps/web",
		},
	}

	tests := []struct {
		name    string
		files   map[string]string
		mode    string
		wantErr bool
	}{
		{
			name: "report_only_present_in_project_dir",
			files: map[string]string{
				"apps/web/package.json": `{"name":"web"}`,
			},
			mode:    PruneModeReportOnly,
			wantErr: false,
		},
		{
			name: "report_only_missing_in_project_dir",
			files: map[string]string{
				"package.json": `{"name":"root"}`,
				"apps/web/.next/standalone/apps/web/package.json": `{"name":"web"}`,
			},
			mode:    PruneModeReportOnly,
			wantErr: true,
		},
		{
			name: "prune_false_present_in_project_dir",
			files: map[string]string{
				"apps/web/package.json": `{"name":"web"}`,
			},
			mode:    PruneModeFalse,
			wantErr: false,
		},
		{
			name: "prune_false_missing_in_project_dir",
			files: map[string]string{
				"package.json": `{"name":"root"}`,
			},
			mode:    PruneModeFalse,
			wantErr: true,
		},
		{
			name: "prune_true_present_in_server_dir",
			files: map[string]string{
				"apps/web/.next/standalone/apps/web/package.json": `{"name":"web"}`,
			},
			mode:    PruneModeTrue,
			wantErr: false,
		},
		{
			name: "prune_true_missing_in_server_dir",
			files: map[string]string{
				"package.json":          `{"name":"root"}`,
				"apps/web/package.json": `{"name":"web"}`,
			},
			mode:    PruneModeTrue,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			writeTestFiles(t, ws, tc.files)
			err := CheckBootPackageJSON(ws, schema, tc.mode)
			if (err != nil) != tc.wantErr {
				t.Errorf("CheckBootPackageJSON(_, _, %q) error = %v, wantErr %v", tc.mode, err, tc.wantErr)
			}
		})
	}
}

func TestCheckProjectFallbackFiles(t *testing.T) {
	const (
		projectPkgJSON  = "apps/web/package.json"
		runtimeManifest = ".cloudrun/next-routing.json"
		reqServerFiles  = "apps/web/.next/required-server-files.json"
	)
	schema := &BundleSchema{
		Metadata: Metadata{
			ProjectDir:      "apps/web",
			DistDir:         "apps/web/.next",
			ServerDir:       "apps/web/.next/standalone/apps/web",
			RuntimeManifest: runtimeManifest,
		},
	}

	tests := []struct {
		name  string
		files map[string]string
		// wantNamed lists the files the error must name. Empty means no error.
		wantNamed []string
	}{
		{
			name: "valid_when_project_package_json_and_manifests_exist",
			files: map[string]string{
				projectPkgJSON:  `{"name":"web"}`,
				runtimeManifest: `{"version":1}`,
				reqServerFiles:  `{"version":1}`,
			},
		},
		{
			name: "project_package_json_missing",
			files: map[string]string{
				"package.json":  `{"name":"root"}`,
				runtimeManifest: `{"version":1}`,
				reqServerFiles:  `{"version":1}`,
			},
			wantNamed: []string{projectPkgJSON},
		},
		{
			name: "runtime_manifest_empty",
			files: map[string]string{
				projectPkgJSON:  `{"name":"web"}`,
				runtimeManifest: `{}`,
				reqServerFiles:  `{"version":1}`,
			},
			wantNamed: []string{runtimeManifest},
		},
		{
			name: "both_manifests_missing",
			files: map[string]string{
				projectPkgJSON: `{"name":"web"}`,
			},
			wantNamed: []string{runtimeManifest, reqServerFiles},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			writeTestFiles(t, ws, tc.files)
			err := CheckProjectFallbackFiles(ws, schema)
			if len(tc.wantNamed) == 0 {
				if err != nil {
					t.Errorf("CheckProjectFallbackFiles() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CheckProjectFallbackFiles() = nil, want error naming %q", tc.wantNamed)
			}
			for _, f := range []string{projectPkgJSON, runtimeManifest, reqServerFiles} {
				if got, want := strings.Contains(err.Error(), f), slices.Contains(tc.wantNamed, f); got != want {
					t.Errorf("CheckProjectFallbackFiles() = %q, names %s: %t, want %t", err, f, got, want)
				}
			}
		})
	}
}

func writeTestFiles(t *testing.T, root string, files map[string]string) {
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
