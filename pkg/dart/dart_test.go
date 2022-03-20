// Copyright 2022 Google LLC
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

package dart

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestResolvePackageVersion(t *testing.T) {
	testCases := []struct {
		name       string
		env        string
		httpStatus int
		response   string
		want       string
		wantError  bool
	}{
		{
			name: "from env",
			env:  "2.14.0",
			want: "2.14.0",
		},
		{
			name: "fetched version",
			response: `{
				"date": "2022-02-08",
				"version": "2.16.1",
				"revision": "0180af250ff518cc0fa494a4eb484ce11ec1e62c"
			}`,
			want: "2.16.1",
		},
		{
			name:       "bad response code",
			httpStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(tc.response),
				testserver.WithMockURL(&versionURL),
			)

			if tc.env != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.env)
			}

			got, err := DetectSDKVersion()
			if tc.wantError == (err == nil) {
				t.Errorf(`DetectSDKVersion() got error: %v, want error?: %v`, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf(`DetectSDKVersion() = %q, want %q`, got, tc.want)
			}
		})
	}
}

func TestHasBuildRunner(t *testing.T) {
	testCases := []struct {
		name    string
		pubspec string
		want    bool
		wantErr bool
	}{
		{
			name: "no pubspec.yaml",
		},
		{
			name:    "no dependencies",
			pubspec: `name: test`,
		},
		{
			name: "no build_runner",
			pubspec: `
name: example_json_function

dependencies:
  functions_framework: ^0.4.0

dev_dependencies:
  functions_framework_builder: ^0.4.0
`,
		},
		{
			name: "with dev_dependency",
			pubspec: `
name: example_json_function

dependencies:
  functions_framework: ^0.4.0

dev_dependencies:
  build_runner: ^2.0.0
`,
			want: true,
		},
		{
			name: "with dev_dependency",
			pubspec: `
name: example_json_function

dependencies:
  build_runner: ^2.0.0

dev_dependencies:
  functions_framework: ^0.4.0
`,
			want: true,
		},
		{
			name:    "invalid yaml",
			pubspec: "\t",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.pubspec != "" {
				path := filepath.Join(dir, "pubspec.yaml")
				if err := os.WriteFile(path, []byte(tc.pubspec), 0744); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}
			got, err := HasBuildRunner(dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("HasBuildRunner(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("HasBuildRunner(%q) = %t, want %t", dir, got, tc.want)
			}
		})
	}
}
