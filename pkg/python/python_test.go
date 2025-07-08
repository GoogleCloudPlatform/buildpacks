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
		wantMatch      bool
		wantMatchErr   bool
		stackID        string
	}{
		{
			name: "default_to_latest_for_default_stack_ubuntu1804_is_default_for_unit_tests",
			want: "3.9.*",
		},
		{
			name:    "version_from_GOOGLE_PYTHON_VERSION",
			version: "3.8.0",
			want:    "3.8.0",
		},
		{
			name:           "version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.8.0",
			want:           "3.8.0",
		},
		{
			name:           "GOOGLE_PYTHON_VERSION_take_precedence_over_GOOGLE_RUNTIME_VERSION",
			version:        "3.8.0",
			runtimeVersion: "3.8.1",
			want:           "3.8.0",
		},
		{
			name:        "version_from_.python-version_file",
			versionFile: "3.8.0",
			want:        "3.8.0",
		},
		{
			name:        "empty_.python-version_file",
			versionFile: " ",
			wantErr:     true,
		},
		{
			name:           "GOOGLE_RUNTIME_VERSION_take_precedence_over_.python-version",
			runtimeVersion: "3.8.0",
			versionFile:    "3.8.1",
			want:           "3.8.0",
		},
		{
			name:           "version_above_3.13.0_through_runtime_version",
			runtimeVersion: "3.13.1",
			want:           "3.13.1",
		},
		{
			name:    "version_below_3.13.0",
			version: "3.12.1",
			want:    "3.12.1",
		},
		{
			name:    "version_with_prerelease",
			version: "3.14.0a1",
			want:    "3.14.0a1",
		},
		{
			name:    "version_with_RC",
			version: "3.13.0rc1",
			want:    "3.13.0rc1",
		},
		{
			name:    "No_version_but_stackID_is_google.22",
			stackID: "google.22",
			want:    "3.13.*",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))
			if tc.stackID != "" {
				ctx = gcp.NewContext(gcp.WithApplicationRoot(dir), gcp.WithStackID(tc.stackID))
			}

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

func TestVersionMatchesSemver(t *testing.T) {
	testCases := []struct {
		name         string
		versionRange string
		version      string
		want         bool
		wantErr      bool
	}{
		{
			name:         "version_matches_semver_range",
			versionRange: ">=3.13.0",
			version:      "3.13.1",
			want:         true,
		},
		{
			name:         "version_does_not_match_semver_range",
			versionRange: ">=3.13.0",
			version:      "3.12.1",
			want:         false,
		},
		{
			name:         "invalid_version_range",
			versionRange: "3.13.0",
			version:      "3.12.1",
			want:         false,
		},
		{
			name:         "invalid_version",
			versionRange: ">=3.13.0",
			version:      "3.12.1a",
			wantErr:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			got, err := versionMatchesSemver(ctx, tc.versionRange, tc.version)
			if tc.wantErr == (err == nil) {
				t.Errorf("versionMatchesSemver(ctx, %q, %q) got error: %v, want err? %t", tc.versionRange, tc.version, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("versionMatchesSemver(ctx, %q, %q) = %t, want %t", tc.versionRange, tc.version, got, tc.want)
			}
		})
	}
}

func TestSupportSmartDefaultEntrypoint(t *testing.T) {
	testCases := []struct {
		name           string
		version        string
		runtimeVersion string
		versionFile    string
		stackID        string
		want           bool
		wantErr        bool
	}{
		{
			name: "default_to_latest_for_default_stack_ubuntu1804_is_default_for_unit_tests",
			want: false,
		},
		{
			name:    "supported_version_from_GOOGLE_PYTHON_VERSION",
			version: "3.14.0",
			want:    true,
		},
		{
			name:    "unsupported_version_from_GOOGLE_PYTHON_VERSION",
			version: "3.8.0",
			want:    false,
		},
		{
			name:           "unsupported_version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.8.0",
			want:           false,
		},
		{
			name:           "supported_version_from_GOOGLE_RUNTIME_VERSION",
			runtimeVersion: "3.13.8",
			want:           true,
		},
		{
			name:        "empty_.python-version_file",
			versionFile: " ",
			wantErr:     true,
		},
		{
			name:    "version_above_3.13.0",
			version: "3.13.1",
			want:    true,
		},
		{
			name:    "version_below_3.13.0",
			version: "3.12.1",
			want:    false,
		},
		{
			name:    "version_with_prerelease",
			version: "3.14.0a1",
			wantErr: true,
			want:    false, // We don't support prerelease versions. Modify once we add support for prerelease versions.
		},
		{
			name:    "version_with_RC",
			version: "3.13.0rc1",
			wantErr: true,
			want:    false, // We don't support RC versions. Modify once we add support for RC versions.
		},
		{
			name:    "No_version_but_stackID_is_google.22",
			stackID: "google.22",
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))
			if tc.stackID != "" {
				ctx = gcp.NewContext(gcp.WithApplicationRoot(dir), gcp.WithStackID(tc.stackID))
			}

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

			boolGot, err := SupportsSmartDefaultEntrypoint(ctx)
			if tc.wantErr == (err == nil) {
				t.Errorf("SupportsSmartDefaultEntrypoint(ctx, %q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if boolGot != tc.want {
				t.Errorf("SupportsSmartDefaultEntrypoint(ctx, %q) = %t, want %t", dir, boolGot, tc.want)
			}
		})
	}
}
