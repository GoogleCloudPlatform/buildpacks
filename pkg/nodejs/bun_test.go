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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestDetectBunVersion(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		wantVersion string
		wantError   bool
	}{
		{
			name:        "no_package.json_returns_latest",
			packageJSON: PackageJSON{},
			npmResponse: `{
        "name": "bun",
        "dist-tags": {
          "latest": "1.0.0"
        },
        "versions": {
          "1.0.0": {
            "name": "bun",
            "version": "1.0.0"
          }
        },
        "modified": "2022-01-27T21:10:55.626Z"
      }`,
			wantVersion: "1.0.0",
		},
		{
			name: "only_engines_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: "1.0.0",
				},
			},
			wantVersion: "1.0.0",
		},
		{
			name: "only_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "bun@1.0.0",
			},
			wantVersion: "1.0.0",
		},
		{
			name: "both_engine_and_packageManager_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					Bun: "1.0.0",
				},
				PackageManager: "bun@0.9.0",
			},
			wantVersion: "1.0.0",
		},
		{
			name: "invalid_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "yarn@1.0.0",
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
