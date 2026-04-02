// Copyright 2023 Google LLC
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
package nodejs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/tooling"
	"github.com/buildpacks/libcnb/v2"
)

func TestInstallPNPM(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		wantFile    string
		wantError   bool
	}{
		{
			name:     "no version constraint",
			wantFile: "bin/pnpm",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{},
		},
		{
			name:     "valid version constraint",
			wantFile: "bin/pnpm",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.x.x",
				},
			},
		},
		{
			name: "invalid version",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: ">9.0.0",
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithJSON(`pnpm!`),
				testserver.WithMockURL(&pnpmDownloadURL),
			)
			testserver.New(
				t,
				testserver.WithJSON(tc.npmResponse),
				testserver.WithMockURL(&npmRegistryURL),
			)

			layer := &libcnb.Layer{
				Name:     "pnpm_test",
				Path:     t.TempDir(),
				Metadata: map[string]any{},
			}
			err := InstallPNPM(gcpbuildpack.NewContext(), layer, &tc.packageJSON)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallPNPM() got error: %v, want error? %v", err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(layer.Path, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Missing file: %s (%v)", fp, err)
				}
			}
		})
	}
}
func TestDetectPNPMVersion(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		stackID     string
		wantVersion string
		wantError   bool
	}{
		{
			name:        "no_package.json_returns_pinned_version_from_tooling_bzl",
			packageJSON: PackageJSON{},
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "9.2.0"
				},
				"versions": {
					"9.2.0": {
						"name": "npm",
						"version": "9.2.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			stackID:     "ubuntu1804",
			wantVersion: "10.12.4", // pinned version for ubuntu1804
		},
		{
			name:        "no_package.json_returns_latest_version_from_tooling_bzl",
			packageJSON: PackageJSON{},
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "9.2.0"
				},
				"versions": {
					"9.2.0": {
						"name": "npm",
						"version": "9.2.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			stackID:     "ubuntu2204",
			wantVersion: "10.32.1",
		},
		{
			name: "only_engines_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.2.0",
				},
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "only_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "pnpm@8.2.0",
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "both_engine_and_packageManager_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.2.0",
				},
				PackageManager: "pnpm@8.1.0",
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "invalid_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "yarn@8.2.0",
			},
			stackID:   "ubuntu2204",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithJSON(tc.npmResponse),
				testserver.WithMockURL(&npmRegistryURL),
			)

			ctx := gcpbuildpack.NewContext()
			defer tooling.MockData()()

			version, err := detectPNPMVersion(ctx, &tc.packageJSON, tc.stackID)
			if version != tc.wantVersion {
				t.Errorf("detectPNPMVersion() got version: %v, want version: %v", version, tc.wantVersion)
			}
			if tc.wantError == (err == nil) {
				t.Fatalf("detectPNPMVersion() got error: %v, want error? %v", err, tc.wantError)
			}
		})
	}
}
