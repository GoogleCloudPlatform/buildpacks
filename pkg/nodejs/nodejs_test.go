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
	"io/ioutil"
	"os"
	"path/filepath"
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
		Scripts: packageScriptsJSON{
			Start: "my-start",
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
			defer func(fn func(*gcp.Context) string) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcp.Context) string { return tc.version }

			home := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(home))

			if tc.packageJSON != "" {
				pkgJSON := filepath.Join(home, "package.json")
				if err := os.WriteFile(pkgJSON, []byte(tc.packageJSON), os.FileMode(0744)); err != nil {
					t.Fatalf("writing file %q: %v", pkgJSON, err)
				}
			}

			got, err := SkipSyntaxCheck(ctx, tc.filePath)
			if err != nil {
				t.Fatalf("Node.js %v: SkipSyntaxCheck(ctx, %q) got error: %v", tc.version, tc.filePath, err)
			}
			if got != tc.want {
				t.Errorf("Node.js %v: SkipSyntaxCheck(ctx, %q) = %t, want %t", tc.version, tc.filePath, got, tc.want)
			}
		})
	}
}

func TestHasGCPBuild(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON string
		want        bool
		wantErr     bool
	}{
		{
			name: "no package.json",
			want: false,
		},
		{
			name:        "invalid package.json",
			packageJSON: `invalid json`,
			wantErr:     true,
		},
		{
			name:        "no gcp-build",
			packageJSON: `{}`,
			want:        false,
		},
		{
			name:        "with gcp-build",
			packageJSON: `{"scripts": {"gcp-build": "ls"}}`,
			want:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			dir := t.TempDir()
			if tc.packageJSON != "" {
				path := filepath.Join(dir, "package.json")
				if err := ioutil.WriteFile(path, []byte(tc.packageJSON), 0744); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}

			got, err := HasGCPBuild(dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("HasGCPBuild(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("HasGCPBuild(%q) = %t, want %t", dir, got, tc.want)
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
