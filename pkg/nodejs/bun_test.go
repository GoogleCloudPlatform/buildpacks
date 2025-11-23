// Copyright 2025 Google LLC
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
	"github.com/buildpacks/libcnb/v2"
)

func TestInstallBun(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		wantFile    string
		wantError   bool
	}{
		{
			name:     "no version constraint",
			wantFile: "bin/bun",
			npmResponse: `{
				"name": "bun",
				"dist-tags": {
					"latest": "1.0.21"
				},
				"versions": {
					"1.0.21": {
						"name": "bun",
						"version": "1.0.21"
					}
				},
				"modified": "2024-01-15T10:00:00.000Z"
			}`,
			packageJSON: PackageJSON{},
		},
		{
			name:     "valid version constraint from engines",
			wantFile: "bin/bun",
			npmResponse: `{
				"name": "bun",
				"dist-tags": {
					"latest": "1.0.21"
				},
				"versions": {
					"1.0.20": {
						"name": "bun",
						"version": "1.0.20"
					},
					"1.0.21": {
						"name": "bun",
						"version": "1.0.21"
					}
				},
				"modified": "2024-01-15T10:00:00.000Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: "1.0.x",
				},
			},
		},
		{
			name: "invalid version",
			npmResponse: `{
				"name": "bun",
				"dist-tags": {
					"latest": "1.0.21"
				},
				"versions": {
					"1.0.21": {
						"name": "bun",
						"version": "1.0.21"
					}
				},
				"modified": "2024-01-15T10:00:00.000Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: ">2.0.0",
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock the Bun download URL (GitHub releases)
			testserver.New(
				t,
				testserver.WithFile("bun-linux-x64/bun", []byte("fake bun binary")),
				testserver.WithZip(),
				testserver.WithMockURL(&bunDownloadURL),
			)
			// Mock the npm registry for version resolution
			testserver.New(
				t,
				testserver.WithJSON(tc.npmResponse),
				testserver.WithMockURL(&npmRegistryURL),
			)

			layer := &libcnb.Layer{
				Name:     "bun_test",
				Path:     t.TempDir(),
				Metadata: map[string]any{},
			}
			err := InstallBun(gcpbuildpack.NewContext(), layer, &tc.packageJSON)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallBun() got error: %v, want error? %v", err, tc.wantError)
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

func TestDetectBunVersion(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		wantVersion string
		wantError   bool
	}{
		{
			name:        "no package.json returns latest",
			packageJSON: PackageJSON{},
			npmResponse: `{
				"name": "bun",
				"dist-tags": {
					"latest": "1.0.21"
				},
				"versions": {
					"1.0.21": {
						"name": "bun",
						"version": "1.0.21"
					}
				},
				"modified": "2024-01-15T10:00:00.000Z"
			}`,
			wantVersion: "1.0.21",
		},
		{
			name: "only engines version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: "1.0.20",
				},
			},
			wantVersion: "1.0.20",
		},
		{
			name: "only packageManager version",
			packageJSON: PackageJSON{
				PackageManager: "bun@1.0.20",
			},
			wantVersion: "1.0.20",
		},
		{
			name: "both engines and packageManager version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: "1.0.20",
				},
				PackageManager: "bun@1.0.19",
			},
			wantVersion: "1.0.20", // engines takes precedence
		},
		{
			name: "invalid packageManager version",
			packageJSON: PackageJSON{
				PackageManager: "pnpm@1.0.20",
			},
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

			version, err := detectBunVersion(&tc.packageJSON)
			if version != tc.wantVersion {
				t.Errorf("detectBunVersion() got version: %v, want version: %v", version, tc.wantVersion)
			}
			if tc.wantError == (err == nil) {
				t.Fatalf("detectBunVersion() got error: %v, want error? %v", err, tc.wantError)
			}
		})
	}
}
