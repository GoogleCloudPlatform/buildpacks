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
	"reflect"
	"testing"

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
		want        string
		wantErr     bool
	}{
		{
			name: "default is empty",
			want: defaultVersionConstraint,
		},
		{
			name:        "package.json without engines",
			packageJSON: `{}`,
			want:        defaultVersionConstraint,
		},
		{
			name:    "GOOGLE_NODEJS_VERSION is set",
			nodeEnv: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:       "GOOGLE_RUNTIME_VERSION is set",
			runtimeEnv: "3.3.3",
			want:       "3.3.3",
		},
		{
			name:       "GOOGLE_NODEJS_VERSION and GOOGLE_RUNTIME_VERSION set",
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
			name:        "GOOGLE_RUNTIME_VERSION and engines.nodejs set",
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tc.rootDir))
			got, err := ReadNodeDependencies(ctx, tc.appDir)
			if err != nil && !tc.expectedError {
				t.Fatalf("ReadNodeDependencies returned an unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ReadNodeDependencies package.json\ngot %#v\nwant %#v", got, want)
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
			wantVersion: "14.1.4",
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
