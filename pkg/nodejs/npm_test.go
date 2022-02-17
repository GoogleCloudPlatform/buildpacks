// Copyright 2021 Google LLC
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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestNPMInstallCommand(t *testing.T) {
	testCases := []struct {
		version string
		want    string
	}{
		{
			version: "v10.1.1",
			want:    "install",
		},
		{
			version: "v8.17.0",
			want:    "install",
		},
		{
			version: "v15.11.0",
			want:    "ci",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) string) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcpbuildpack.Context) string { return tc.version }

			got, err := NPMInstallCommand(nil)
			if err != nil {
				t.Fatalf("Node.js %v: NPMInstallCommand(nil) got error: %v", tc.version, err)
			}
			if got != tc.want {
				t.Errorf("Node.js %v: NPMInstallCommand(nil) = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}

func TestSupportsNPMPrune(t *testing.T) {
	testCases := []struct {
		version string
		want    bool
	}{
		{
			version: "8.3.1",
			want:    true,
		},
		{
			version: "5.7.0",
			want:    true,
		},
		{
			version: "5.0.1",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) string) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) string { return tc.version }

			got, err := SupportsNPMPrune(nil)
			if err != nil {
				t.Errorf("npm %v: SupportsNPMPrune(nil) got error: %v", tc.version, err)
			}
			if got != tc.want {
				t.Errorf("npm %v: SupportsNPMPrune(nil) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}
