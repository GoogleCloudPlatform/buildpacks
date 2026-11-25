// Copyright 2020 Google LLC
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
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/google/go-cmp/cmp"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestReadPackageJSONIfExists(t *testing.T) {
	want := PackageJSON{
		Engines: packageEnginesJSON{
			Node: "my-node",
			NPM:  "my-npm",
		},
		Scripts: map[string]string{
			"start": "my-start",
		},
		Dependencies: map[string]string{
			"a": "1.0",
			"b": "2.0",
		},
		DevDependencies: map[string]string{
			"c": "3.0",
		},
	}

	got, err := ReadPackageJSONIfExists(testdata.MustGetPath("testdata/test-read-package/"))
	if err != nil {
		t.Fatalf("ReadPackageJSONIfExists got error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadPackageJSONIfExists\ngot %#v\nwant %#v", *got, want)
	}
}

func TestReadPackageJSONIfExistsDoesNotExist(t *testing.T) {
	got, err := ReadPackageJSONIfExists(t.TempDir())
	if err != nil {
		t.Fatalf("ReadPackageJSONIfExists got error: %v", err)
	}
	if got != nil {
		t.Errorf("ReadPackageJSONIfExists\ngot %#v\nwant nil", *got)
	}
}

func TestSkipSyntaxCheck(t *testing.T) {
	testCases := []struct {
		name        string
		version     string
		packageJSON string
		filePath    string
		want        bool
	}{
		{
			name:        "Node.js 14",
			version:     "v14.1.1",
			packageJSON: `{"type": "module"}`,
			filePath:    "index.mjs",
			want:        false,
		},
		{
			name:     "Node.js 16 with mjs",
			version:  "v16.1.1",
			filePath: "index.mjs",
			want:     true,
		},
		{
			name:        "Node.js 16 with modules",
			version:     "v16.1.1",
			packageJSON: `{"type": "module"}`,
			want:        true,
		},
		{
			name:    "Node.js 16 without ESM",
			version: "v16.1.1",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func(fn func(*gcp.Context) (string, error)) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcp.Context) (string, error) { return tc.version, nil }

			home := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(home))

			var pjs *PackageJSON
			if tc.packageJSON != "" {
				if err := json.Unmarshal([]byte(tc.packageJSON), &pjs); err != nil {
					t.Errorf("failed to unmarshal package.json: %q, err: %v", tc.packageJSON, err)
				}
			}

			got, err := SkipSyntaxCheck(ctx, tc.filePath, pjs)
			if err != nil {
				t.Fatalf("Node.js %v: SkipSyntaxCheck(ctx, %q) got error: %v", tc.version, tc.filePath, err)
			}
			if got != tc.want {
				t.Errorf("Node.js %v: SkipSyntaxCheck(ctx, %q) = %t, want %t", tc.version, tc.filePath, got, tc.want)
			}
		})
	}
}
func TestHasApphostingBuild(t *testing.T) {
	tests := []struct {
		name             string
		packageJSON      *PackageJSON
		apphostingSchema apphostingschema.AppHostingSchema
		want             bool
	}{
		{
			name: "package.json has apphosting:build script",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					"apphosting:build": "some command",
				},
			},
			want: true,
		},
		{
			name: "apphosting schema has build command",
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{
					BuildCommand: "some command",
				},
			},
			want: true,
		},
		{
			name: "both package.json and schema have build command",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					ScriptApphostingBuild: "some command",
				},
			},
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{
					BuildCommand: "another command",
				},
			},
			want: true,
		},
		{
			name: "neither package.json nor schema has build command",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					"some other script": "some command",
				},
			},
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{},
			},
			want: false,
		},
		{
			name:             "nil package.json and nil schema scripts",
			packageJSON:      nil,
			apphostingSchema: apphostingschema.AppHostingSchema{},
			want:             false,
		},
		{
			name:        "nil PackageJson, valid schema",
			packageJSON: nil,
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{
					BuildCommand: "some command",
				},
			},
			want: true,
		},
		{
			name: "valid packageJson, nil schema",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					ScriptApphostingBuild: "some command",
				},
			},
			apphostingSchema: apphostingschema.AppHostingSchema{},
			want:             true,
		},
		{
			name:        "nil PackageJson, empty schema scripts",
			packageJSON: nil,
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{},
			},
			want: false,
		},
		{
			name: "valid PackageJson, empty schema scripts",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					"some other script": "some command",
				},
			},
			apphostingSchema: apphostingschema.AppHostingSchema{
				Scripts: apphostingschema.Scripts{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasApphostingPackageOrYamlBuild(tt.packageJSON, tt.apphostingSchema)
			if got != tt.want {
				t.Errorf("HasApphostingBuild() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestHasScript(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON *PackageJSON
		script      string
		want        bool
	}{
		{
			name:        "nil package.json",
			packageJSON: nil,
			want:        false,
		},
		{
			name: "matching script",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					"gcp-build": "my-script",
				},
			},
			script: "gcp-build",
			want:   true,
		},
		{
			name: "mismatching script",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{
					"gcp-build": "my-script",
				},
			},
			script: "build",
			want:   false,
		},
		{
			name: "matching empty script",
			packageJSON: &PackageJSON{
				Scripts: map[string]string{"gcp-build": ""},
			},
			script: "gcp-build",
			want:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasScript(tc.packageJSON, tc.script)
			if got != tc.want {
				t.Errorf("HasGCPBuild(%v) = %t, want %t", tc.packageJSON, got, tc.want)
			}
		})
	}
}

func TestHasDevDependencies(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON *PackageJSON
		want        bool
	}{
		{
			name:        "nil package",
			packageJSON: nil,
			want:        false,
		},
		{
			name: "has",
			packageJSON: &PackageJSON{
				DevDependencies: map[string]string{
					"my": "dep",
				},
			},
			want: true,
		},
		{
			name:        "does not have",
			packageJSON: &PackageJSON{},
			want:        false,
		},
		{
			name: "empty",
			packageJSON: &PackageJSON{
				DevDependencies: map[string]string{},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasDevDependencies(tc.packageJSON)
			if got != tc.want {
				t.Errorf("HasDevDependencies(%v) = %t, want %t", tc.packageJSON, got, tc.want)
			}
		})
	}
}

func TestDependencyVersion(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON *PackageJSON
		dependency  string
		want        string
	}{
		{
			name:        "nil package.json",
			packageJSON: nil,
			want:        "",
		},
		{
			name: "matching dependency",
			packageJSON: &PackageJSON{
				Dependencies: map[string]string{
					"genkit": "1.0.0",
				},
			},
			dependency: "genkit",
			want:       "1.0.0",
		},
		{
			name: "mismatching dependency",
			packageJSON: &PackageJSON{
				Dependencies: map[string]string{
					"genkit": "1.0.0",
				},
			},
			dependency: "@google/generative-ai",
			want:       "",
		},
		{
			name: "empty dependencies",
			packageJSON: &PackageJSON{
				Dependencies: map[string]string{},
			},
			dependency: "genkit",
			want:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := DependencyVersion(tc.packageJSON, tc.dependency)
			if got != tc.want {
				t.Errorf("DependencyVersion(%v) = %s, want %s", tc.packageJSON, got, tc.want)
			}
		})
	}
}

func TestRequestedNodejsVersion(t *testing.T) {
	testCases := []struct {
		name        string
		nodeEnv     string
		runtimeEnv  string
		packageJSON string
		stackID     string
		want        string
		wantErr     bool
	}{
		{
			name: "default_is_empty",
			want: "26.*.*",
		},
		{
			name:        "package.json_without_engines",
			packageJSON: `{}`,
			want:        "26.*.*",
		},
		{
			name:    "default_on_ubuntu_24_stack",
			stackID: "google.24.full",
			want:    "26.*.*",
		},
		{
			name:    "GOOGLE_NODEJS_VERSION_is_set",
			nodeEnv: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:       "GOOGLE_RUNTIME_VERSION_is_set",
			runtimeEnv: "3.3.3",
			want:       "3.3.3",
		},
		{
			name:       "GOOGLE_NODEJS_VERSION_and_GOOGLE_RUNTIME_VERSION_set",
			nodeEnv:    "1.2.3",
			runtimeEnv: "3.3.3",
			want:       "1.2.3",
		},
		{
			name:        "engines.nodejs",
			packageJSON: `{"engines": {"node": "2.2.2"}}`,
			want:        "2.2.2",
		},
		{
			name:        "GOOGLE_RUNTIME_VERSION_and_engines.nodejs_set",
			packageJSON: `{"engines": {"node": "2.2.2"}}`,
			runtimeEnv:  "3.3.3",
			want:        "3.3.3",
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
			if tc.nodeEnv != "" {
				t.Setenv("GOOGLE_NODEJS_VERSION", tc.nodeEnv)
			}
			if tc.runtimeEnv != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.runtimeEnv)
			}

			ctx := gcp.NewContext()
			if tc.stackID != "" {
				ctx = gcp.NewContext(gcp.WithStackID(tc.stackID))
			}
			got, err := RequestedNodejsVersion(ctx, pjs)
			if tc.wantErr == (err == nil) {
				t.Errorf("RequestedNodejsVersion(ctx, %q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RequestedNodejsVersion(ctx, %q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestIsNodeJS8Runtime(t *testing.T) {
	testCases := []struct {
		name           string
		runtimeEnvVar  string
		expectedResult bool
	}{
		{
			name:           "empty should return false",
			runtimeEnvVar:  "",
			expectedResult: false,
		},
		{
			name:           "go111 should return false",
			runtimeEnvVar:  "go111",
			expectedResult: false,
		},
		{
			name:           "nodejs8 should return true",
			runtimeEnvVar:  "nodejs8",
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setGoogleRuntime(t, tc.runtimeEnvVar)
			result := IsNodeJS8Runtime()
			if result != tc.expectedResult {
				t.Fatalf("IsNodeJS8Runtime(GOOGLE_RUNTIME=%v) = %v, want %v", tc.runtimeEnvVar, result, tc.expectedResult)
			}
		})
	}
}

func TestReadNodeDependencies(t *testing.T) {

	pjs := &PackageJSON{
		Engines: packageEnginesJSON{
			Node: "my-node",
			NPM:  "my-npm",
		},
		Scripts: map[string]string{
			"start": "my-start",
		},
		Dependencies: map[string]string{
			"a": "1.0",
			"b": "2.0",
		},
		DevDependencies: map[string]string{
			"c": "3.0",
		},
	}

	want := &NodeDependencies{
		PackageJSON:  pjs,
		LockfilePath: testdata.MustGetPath("testdata/test-read-node-deps/package-lock.json"),
	}

	wantNoLockfile := &NodeDependencies{
		PackageJSON: pjs,
	}

	minimalPJS := `{"name": "test-app"}`
	wantMinimalPJS := &PackageJSON{Name: "test-app"}

	bunLockDir := writeTempAppFiles(t, map[string]string{
		"package.json": minimalPJS,
		"bun.lock":     `{"lockfileVersion": 1}`,
	})
	bunLockbDir := writeTempAppFiles(t, map[string]string{
		"package.json": minimalPJS,
		"bun.lockb":    "binary-lockfile",
	})
	bunMonorepoDir := writeTempAppFiles(t, map[string]string{
		"bun.lock":              `{"lockfileVersion": 1}`,
		"apps/web/package.json": minimalPJS,
	})
	bunAppOverRootDir := writeTempAppFiles(t, map[string]string{
		"package-lock.json":     `{"lockfileVersion": 3}`,
		"apps/web/package.json": minimalPJS,
		"apps/web/bun.lock":     `{"lockfileVersion": 1}`,
	})
	pnpmPrecedenceDir := writeTempAppFiles(t, map[string]string{
		"package.json":        minimalPJS,
		"pnpm-lock.yaml":      "lockfileVersion: '6.0'",
		"yarn.lock":           "# yarn lockfile v1",
		"npm-shrinkwrap.json": `{"lockfileVersion": 3}`,
		"package-lock.json":   `{"lockfileVersion": 3}`,
		"bun.lock":            `{"lockfileVersion": 1}`,
		"bun.lockb":           "binary-lockfile",
	})
	yarnPrecedenceDir := writeTempAppFiles(t, map[string]string{
		"package.json":        minimalPJS,
		"yarn.lock":           "# yarn lockfile v1",
		"npm-shrinkwrap.json": `{"lockfileVersion": 3}`,
		"package-lock.json":   `{"lockfileVersion": 3}`,
		"bun.lock":            `{"lockfileVersion": 1}`,
		"bun.lockb":           "binary-lockfile",
	})
	shrinkwrapPrecedenceDir := writeTempAppFiles(t, map[string]string{
		"package.json":        minimalPJS,
		"npm-shrinkwrap.json": `{"lockfileVersion": 3}`,
		"package-lock.json":   `{"lockfileVersion": 3}`,
		"bun.lock":            `{"lockfileVersion": 1}`,
		"bun.lockb":           "binary-lockfile",
	})
	npmPrecedenceDir := writeTempAppFiles(t, map[string]string{
		"package.json":      minimalPJS,
		"package-lock.json": `{"lockfileVersion": 3}`,
		"bun.lock":          `{"lockfileVersion": 1}`,
		"bun.lockb":         "binary-lockfile",
	})
	bunPrecedenceDir := writeTempAppFiles(t, map[string]string{
		"package.json": minimalPJS,
		"bun.lock":     `{"lockfileVersion": 1}`,
		"bun.lockb":    "binary-lockfile",
	})

	testCases := []struct {
		name          string
		rootDir       string
		appDir        string
		expectedError bool
		want          *NodeDependencies
	}{
		{
			name:          "missing lockfile",
			rootDir:       testdata.MustGetPath("testdata/test-read-package/"),
			appDir:        testdata.MustGetPath("testdata/test-read-package/"),
			expectedError: false,
			want:          wantNoLockfile,
		},
		{
			name:    "package json and lockfile in same dir",
			rootDir: testdata.MustGetPath("testdata/test-read-node-deps/"),
			appDir:  testdata.MustGetPath("testdata/test-read-node-deps/"),
			want:    want,
		},
		{
			name:    "lockfile in root dir and package json in app dir, find lockfile in root dir",
			rootDir: testdata.MustGetPath("testdata/test-read-node-deps-nested/"),
			appDir:  testdata.MustGetPath("testdata/test-read-node-deps-nested/package-a/"),
			want: &NodeDependencies{
				PackageJSON:  want.PackageJSON,
				LockfilePath: testdata.MustGetPath("testdata/test-read-node-deps-nested/package-lock.json"),
			},
		},
		{
			name:    "bun_lock_in_app_dir",
			rootDir: bunLockDir,
			appDir:  bunLockDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(bunLockDir, "bun.lock"),
			},
		},
		{
			name:    "bun_lockb_in_app_dir",
			rootDir: bunLockbDir,
			appDir:  bunLockbDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(bunLockbDir, "bun.lockb"),
			},
		},
		{
			name:    "bun_lock_in_root_dir_when_app_dir_is_subdir",
			rootDir: bunMonorepoDir,
			appDir:  filepath.Join(bunMonorepoDir, "apps/web"),
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(bunMonorepoDir, "bun.lock"),
			},
		},
		{
			name:    "bun_lock_in_app_dir_preferred_over_root_lockfile",
			rootDir: bunAppOverRootDir,
			appDir:  filepath.Join(bunAppOverRootDir, "apps/web"),
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(bunAppOverRootDir, "apps/web/bun.lock"),
			},
		},
		{
			name:    "lockfile_precedence_pnpm_over_all",
			rootDir: pnpmPrecedenceDir,
			appDir:  pnpmPrecedenceDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(pnpmPrecedenceDir, "pnpm-lock.yaml"),
			},
		},
		{
			name:    "lockfile_precedence_yarn_over_npm_and_bun",
			rootDir: yarnPrecedenceDir,
			appDir:  yarnPrecedenceDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(yarnPrecedenceDir, "yarn.lock"),
			},
		},
		{
			name:    "lockfile_precedence_shrinkwrap_over_package_lock_and_bun",
			rootDir: shrinkwrapPrecedenceDir,
			appDir:  shrinkwrapPrecedenceDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(shrinkwrapPrecedenceDir, "npm-shrinkwrap.json"),
			},
		},
		{
			name:    "lockfile_precedence_package_lock_over_bun",
			rootDir: npmPrecedenceDir,
			appDir:  npmPrecedenceDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(npmPrecedenceDir, "package-lock.json"),
			},
		},
		{
			name:    "lockfile_precedence_bun_lock_over_bun_lockb",
			rootDir: bunPrecedenceDir,
			appDir:  bunPrecedenceDir,
			want: &NodeDependencies{
				PackageJSON:  wantMinimalPJS,
				LockfilePath: filepath.Join(bunPrecedenceDir, "bun.lock"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tc.rootDir))
			got, err := ReadNodeDependencies(ctx, tc.appDir)
			if gotErr := err != nil; gotErr != tc.expectedError {
				t.Fatalf("ReadNodeDependencies(%q) error = %v, wantErr %t", tc.appDir, err, tc.expectedError)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ReadNodeDependencies(%q) mismatch (-want +got):\n%s", tc.appDir, diff)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	testCases := []struct {
		name        string
		nodeDeps    *NodeDependencies
		wantVersion string
		pkg         string
		wantErr     bool
	}{
		{
			name: "Parses package-lock.json version nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/nextjs-package-lock.json"),
			},
			wantVersion: "15.0.0",
		},
		{
			name: "Parses package-lock version angular",
			pkg:  "angular",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/angular-package-lock.json"),
			},
			wantVersion: "17.1.3",
		},
		{
			name: "Parses yarn.lock berry version nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				PackageJSON: &PackageJSON{
					Dependencies: map[string]string{
						"next": "14.1.4",
					},
				},
				LockfilePath: testdata.MustGetPath("testdata/lock-files/berry-yarn.lock"),
			},
			wantVersion: "14.1.4",
		},
		{
			name: "Parses yarn.lock classic version nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				PackageJSON: &PackageJSON{
					Dependencies: map[string]string{
						"next": "14.1.4",
					},
				},
				LockfilePath: testdata.MustGetPath("testdata/lock-files/classic-yarn.lock"),
			},
			wantVersion: "14.1.4",
		},
		{
			name: "Parses yarn.lock nextjs with other `next` dependencies",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				PackageJSON: &PackageJSON{
					Dependencies: map[string]string{
						"next":            "14.1.4",
						"@prismicio/next": "^2.0.1",
					},
				},
				LockfilePath: testdata.MustGetPath("testdata/lock-files/classic-yarn.lock"),
			},
			wantVersion: "14.1.4",
		},
		{
			name: "Parses yarn.lock for scoped packages",
			pkg:  "@prismicio/next",
			nodeDeps: &NodeDependencies{
				PackageJSON: &PackageJSON{
					Dependencies: map[string]string{
						"next":            "14.1.4",
						"@prismicio/next": "^2.0.1",
					},
				},
				LockfilePath: testdata.MustGetPath("testdata/lock-files/classic-yarn.lock"),
			},
			wantVersion: "2.0.1",
		},
		{
			name: "Parses pnpm-lock v6 version nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/nextjs-v6-pnpm-lock.yaml"),
			},
			wantVersion: "14.1.4",
		},
		{
			name: "Parses pnpm-lock v9 version nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/nextjs-v9-pnpm-lock.yaml"),
			},
			wantVersion: "14.1.4",
		},
		{
			name: "Parses pnpm-lock v6 version angular",
			pkg:  "@angular-devkit/build-angular",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/angular-v6-pnpm-lock.yaml"),
			},
			wantVersion: "17.3.6",
		},
		{
			name: "Parses pnpm-lock v9 version angular",
			pkg:  "@angular-devkit/build-angular",
			nodeDeps: &NodeDependencies{
				LockfilePath: testdata.MustGetPath("testdata/lock-files/angular-v9-pnpm-lock.yaml"),
			},
			wantVersion: "17.3.6",
		},
		{
			name: "bun_lock_nextjs",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				LockfilePath: writeTempLockfile(t, "bun.lock", `{
  // Bun text lockfile with comments and trailing commas
  "lockfileVersion": 1,
  "packages": {
    "next": ["next@16.4.3", "", {}, "sha512-abc",],
    "@next/routing": ["@next/routing@16.4.3", "", {}, "sha512-def",],
  },
}`),
			},
			wantVersion: "16.4.3",
		},
		{
			name: "bun_lock_scoped_package",
			pkg:  "@next/routing",
			nodeDeps: &NodeDependencies{
				LockfilePath: writeTempLockfile(t, "bun.lock", `{
  "lockfileVersion": 1,
  "packages": {
    "@next/routing": ["@next/routing@16.4.3", "", {}, "sha512-def"],
  },
}`),
			},
			wantVersion: "16.4.3",
		},
		{
			name: "bun_lock_missing_package",
			pkg:  "nonexistent",
			nodeDeps: &NodeDependencies{
				LockfilePath: writeTempLockfile(t, "bun.lock", `{
  "lockfileVersion": 1,
  "packages": {
    "next": ["next@16.4.3", "", {}, "sha512-abc"],
  },
}`),
			},
			wantVersion: "",
		},
		{
			name: "bun_lockb_unsupported",
			pkg:  "next",
			nodeDeps: &NodeDependencies{
				LockfilePath: writeTempLockfile(t, "bun.lockb", "binary-lockfile"),
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, err := Version(tc.nodeDeps, tc.pkg)
			if version != tc.wantVersion {
				t.Fatalf("Version(%v, %v) output: %s doesn't match expected output %s", tc.nodeDeps, tc.pkg, version, tc.wantVersion)
			}
			if gotErr := (err != nil); gotErr != tc.wantErr {
				t.Fatalf("Version(_, %v) returned error: %v, wanted: %v, errMsg: %v.", tc.pkg, gotErr, tc.wantErr, err)
			}
		})
	}
}

func TestVersionFromBunLock(t *testing.T) {
	testCases := []struct {
		name    string
		lock    string
		pkg     string
		want    string
		wantErr bool
	}{
		{
			name: "unscoped_package",
			lock: `{"lockfileVersion": 1, "packages": {"next": ["next@16.4.3", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "scoped_package",
			lock: `{"packages": {"@next/routing": ["@next/routing@16.4.3", "", {}, "sha512-def"]}}`,
			pkg:  "@next/routing",
			want: "16.4.3",
		},
		{
			name: "nested_package_under_parent_is_ignored",
			lock: `{"packages": {"send/ms": ["ms@2.1.3", "", {}, "sha512-abc"]}}`,
			pkg:  "ms",
			want: "",
		},
		{
			name: "top_level_entry_ignores_nested_entry",
			lock: `{"packages": {
				"ms": ["ms@2.0.0", "", {}, "sha512-abc"],
				"send/ms": ["ms@2.1.3", "", {}, "sha512-def"]
			}}`,
			pkg:  "ms",
			want: "2.0.0",
		},
		{
			name: "npm_prefixed_version",
			lock: `{"packages": {"foo": ["foo@npm:1.2.3", "", {}, "sha512-abc"]}}`,
			pkg:  "foo",
			want: "1.2.3",
		},
		{
			name: "bare_version_without_package_prefix",
			lock: `{"packages": {"next": ["16.4.3", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "tuple_naming_different_package_is_skipped",
			lock: `{"packages": {"next": ["other@1.0.0", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "object_entry_with_version_field",
			lock: `{"packages": {"next": {"version": "16.0.0"}}}`,
			pkg:  "next",
			want: "16.0.0",
		},
		{
			name: "comments_and_trailing_commas",
			lock: `{
  // Bun lockfile with comments and trailing commas.
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
    "react": ["react@19.2.1", "", {}, "sha512-def",],
  },
}`,
			pkg:  "react",
			want: "19.2.1",
		},
		{
			name: "comment_markers_and_escaped_quotes_inside_strings_are_kept",
			lock: `{
  "packages": {
    "next": ["next@16.4.3", "https://registry.npmjs.org/next/-/next-16.4.3.tgz", { "description": "a \"quoted\" /* not a comment */ value," }, "sha512-//abc"], // trailing comment
  },
}`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "trailing_comma_followed_by_comment",
			lock: `{
  "packages": {
    "next": ["next@16.4.3", "", {}, "sha512-abc"], // last entry
  },
}`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "package_not_in_lockfile",
			lock: `{"packages": {"next": ["next@16.4.3", "", {}, "sha512-abc"]}}`,
			pkg:  "react",
			want: "",
		},
		{
			name: "empty_packages",
			lock: `{"lockfileVersion": 1, "packages": {}}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "no_packages_field",
			lock: `{"lockfileVersion": 1}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "empty_tuple",
			lock: `{"packages": {"next": []}}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "tuple_with_non_string_first_element",
			lock: `{"packages": {"next": [123, "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "tuple_with_empty_first_element",
			lock: `{"packages": {"next": ["", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "",
		},
		{
			name: "trailing_comma_before_closing_bracket",
			lock: `{"packages": {"next": ["next@16.4.3", "", {}, "sha512-abc", ]}}`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "line_comment_at_eof_without_newline",
			lock: `{"packages": {"next": ["next@16.4.3", "", {}, "sha512-abc"]}} // comment at eof`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name: "unclosed_block_comment_after_json",
			lock: `{"packages": {"next": ["next@16.4.3", "", {}, "sha512-abc"]}} /* unclosed`,
			pkg:  "next",
			want: "16.4.3",
		},
		{
			name:    "unclosed_block_comment_mid_object",
			lock:    `{"packages": /* unclosed`,
			pkg:     "next",
			wantErr: true,
		},
		{
			name: "non_semver_workspace_spec_returned_verbatim",
			lock: `{"packages": {"next": ["next@workspace:apps/web"]}}`,
			pkg:  "next",
			want: "workspace:apps/web",
		},
		{
			name: "non_semver_file_spec_returned_verbatim",
			lock: `{"packages": {"next": ["next@file:./local", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "file:./local",
		},
		{
			name: "non_semver_tarball_url_without_at_returned_verbatim",
			lock: `{"packages": {"next": ["https://example.com/next.tgz", "", {}, "sha512-abc"]}}`,
			pkg:  "next",
			want: "https://example.com/next.tgz",
		},
		{
			name:    "malformed_json",
			lock:    `{"packages": {"next": ["next@16.4.3"`,
			pkg:     "next",
			wantErr: true,
		},
		{
			name:    "empty_file",
			lock:    ``,
			pkg:     "next",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := versionFromBunLock([]byte(tc.lock), tc.pkg)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("versionFromBunLock(_, %q) returned error: %v, want error: %t", tc.pkg, err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if got != tc.want {
				t.Errorf("versionFromBunLock(_, %q) = %q, want %q", tc.pkg, got, tc.want)
			}
		})
	}
}

func TestVersionFromBunLockNoPanic(t *testing.T) {
	f := func(raw []byte, pkg string) bool {
		_ = stripJSONC(raw)
		_, _ = versionFromBunLock(raw, pkg)
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Errorf("versionFromBunLock/stripJSONC failed quick.Check: %v", err)
	}
}

func TestOverrideAppHostingBuildScript(t *testing.T) {
	testCases := []struct {
		name                  string
		packageJSONContent    PackageJSON
		apphostingYAMLContent string
		expectedPackageJSON   *PackageJSON
	}{
		{
			name:                  "no package.json, no apphosting.yaml",
			packageJSONContent:    PackageJSON{},
			apphostingYAMLContent: ``,
			expectedPackageJSON:   &PackageJSON{},
		},
		{
			name:               "no package.json, apphosting.yaml with build command",
			packageJSONContent: PackageJSON{},
			apphostingYAMLContent: `scripts:
  buildCommand: custom-build`,
			expectedPackageJSON: &PackageJSON{Scripts: map[string]string{ScriptApphostingBuild: "custom-build"}},
		},
		{
			name:               "package.json with existing scripts, apphosting.yaml with build command",
			packageJSONContent: PackageJSON{Scripts: map[string]string{"test": "echo test"}},
			apphostingYAMLContent: `scripts:
  buildCommand: custom-build`,
			expectedPackageJSON: &PackageJSON{Scripts: map[string]string{"test": "echo test", ScriptApphostingBuild: "custom-build"}},
		},
		{
			name:               "package.json with existing apphosting:build, apphosting.yaml with build command",
			packageJSONContent: PackageJSON{Scripts: map[string]string{ScriptApphostingBuild: "old-build"}},
			apphostingYAMLContent: `scripts:
  buildCommand: custom-build`,
			expectedPackageJSON: &PackageJSON{Scripts: map[string]string{ScriptApphostingBuild: "custom-build"}},
		},
		{
			name:                  "package.json with existing apphosting:build, apphosting.yaml without build command",
			packageJSONContent:    PackageJSON{Scripts: map[string]string{ScriptApphostingBuild: "old-build"}},
			apphostingYAMLContent: ``,
			expectedPackageJSON:   &PackageJSON{Scripts: map[string]string{ScriptApphostingBuild: "old-build"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			packageJSONPath := filepath.Join(tempDir, "package.json")
			preprocessedApphostingPath := filepath.Join(tempDir, "apphosting_preprocessed")
			marshaledJSONSchema, err := json.Marshal(tc.packageJSONContent)
			if err != nil {
				t.Fatalf("failed to marshal apphosting schema: %v", err)
			}
			err = os.WriteFile(packageJSONPath, marshaledJSONSchema, 0644)
			if err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			err = os.WriteFile(preprocessedApphostingPath, []byte(tc.apphostingYAMLContent), 0644)

			if err != nil {
				t.Fatalf("Failed to write apphosting_preprocessed: %v", err)
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempDir))
			result, err := OverrideAppHostingBuildScript(ctx, preprocessedApphostingPath)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(tc.expectedPackageJSON, result); diff != "" {
				t.Errorf("OverrideAppHostingBuildScript() mismatch (-want +got):\n%s", diff)
			}

			if tc.expectedPackageJSON != nil {
				actualPackageJSONContent, err := os.ReadFile(packageJSONPath)
				if err != nil {
					t.Fatalf("Failed to read package.json: %v", err)
				}
				var actualPackageJSON PackageJSON
				err = json.Unmarshal(actualPackageJSONContent, &actualPackageJSON)
				if err != nil {
					t.Fatalf("Failed to unmarshal package.json: %v", err)
				}

				if diff := cmp.Diff(tc.expectedPackageJSON, &actualPackageJSON); diff != "" {
					t.Errorf("package.json file mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
func setGoogleRuntime(t *testing.T, value string) {
	googleRuntimeEnv := "GOOGLE_RUNTIME"
	t.Cleanup(func() {
		if err := os.Unsetenv(googleRuntimeEnv); err != nil {
			t.Fatalf("Error resetting environment variable %q: %v", googleRuntimeEnv, err)
		}
	})
	if err := os.Setenv("GOOGLE_RUNTIME", value); err != nil {
		t.Errorf("Error setting environment variable %q: %v", googleRuntimeEnv, err)
	}
}

func TestEntrypoint(t *testing.T) {
	testCases := []struct {
		name           string
		packageManager string
		capability     bool
		want           []string
	}{
		{
			name:           "without_capability",
			packageManager: "yarn",
			capability:     false,
			want:           []string{"yarn", "run", "start"},
		},
		{
			name:           "with_capability",
			packageManager: "yarn",
			capability:     true,
			want:           []string{"npm", "run", "start"},
		},
		{
			name:           "pnpm_without_capability",
			packageManager: "pnpm",
			capability:     false,
			want:           []string{"pnpm", "run", "start"},
		},
		{
			name:           "pnpm_with_capability",
			packageManager: "pnpm",
			capability:     true,
			want:           []string{"npm", "run", "start"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []gcp.ContextOption
			if tc.capability {
				opts = append(opts, gcp.WithCapability(NPMStartEntrypointCapability, &MakerNPMStartEntrypoint{}))
			}
			ctx := gcp.NewContext(opts...)

			got, err := Entrypoint(ctx, tc.packageManager)
			if err != nil {
				t.Fatalf("Entrypoint(ctx, %q) returned unexpected error: %v", tc.packageManager, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Entrypoint(ctx, %q) returned unexpected difference (-want +got):\n%s", tc.packageManager, diff)
			}
		})
	}
}

func writeTempLockfile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) failed: %v", p, err)
	}
	return p
}

func writeTempAppFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("os.MkdirAll(%q) failed: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) failed: %v", p, err)
		}
	}
	return dir
}
