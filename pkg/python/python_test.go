// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package python

import (
	"os"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestRuntimeVersion(t *testing.T) {
	testCases := []struct {
		name           string
		version        string
		runtimeVersion string
		versionFile    string
		want           string
		wantErr        bool
	}{
		{
			name: "default to *",
			want: "*",
		},
		{
			name:    "version from GOOGLE_PYTHON_VERSION",
			version: "3.8.0",
			want:    "3.8.0",
		},
		{
			name:           "version from GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.8.0",
			want:           "3.8.0",
		},
		{
			name:           "GOOGLE_PYTHON_VERSION take precedence over GOOGLE_RUNTIME_VERSION",
			version:        "3.8.0",
			runtimeVersion: "3.8.1",
			want:           "3.8.0",
		},
		{
			name:        "version from .python-version file",
			versionFile: "3.8.0",
			want:        "3.8.0",
		},
		{
			name:        "empty .python-version file",
			versionFile: " ",
			wantErr:     true,
		},
		{
			name:           "GOOGLE_RUNTIME_VERSION take precedence over .python-version",
			runtimeVersion: "3.8.0",
			versionFile:    "3.8.1",
			want:           "3.8.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))

			if tc.version != "" {
				t.Setenv("GOOGLE_PYTHON_VERSION", tc.version)
			}
			if tc.runtimeVersion != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.runtimeVersion)
			}
			if tc.versionFile != "" {
				versionFile := filepath.Join(dir, ".python-version")
				if err := os.WriteFile(versionFile, []byte(tc.versionFile), os.FileMode(0744)); err != nil {
					t.Fatalf("writing file %q: %v", versionFile, err)
				}
			}

			got, err := RuntimeVersion(ctx, dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("RuntimeVersion(ctx, %q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RuntimeVersion(ctx, %q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}
