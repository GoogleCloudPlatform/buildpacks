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
	"encoding/json"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestRequestedNPMVersion(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON string
		want        string
		wantErr     bool
	}{
		{
			name:        "default is empty",
			packageJSON: `{}`,
			want:        "",
		},
		{
			name:        "engines.npm set",
			packageJSON: `{"engines": {"npm": "2.2.2"}}`,
			want:        "2.2.2",
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

			got, err := RequestedNPMVersion(pjs)
			if tc.wantErr == (err == nil) {
				t.Errorf("RequestedNPMVersion(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RequestedNPMVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestNPMInstallCommand(t *testing.T) {
	testCases := []struct {
		name           string
		npmVersion     string
		nodeVersion    string
		want           string
		targetPlatform string
	}{
		{
			name:       "8.3.1 should return ci",
			npmVersion: "8.3.1",
			want:       "ci",
		},
		{
			name:           "8.3.1 on GAE should return install for backwards compatibility",
			npmVersion:     "8.3.1",
			nodeVersion:    "10.24.1",
			want:           "install",
			targetPlatform: env.TargetPlatformAppEngine,
		},
		{
			name:           "8.3.1 on GCF should return install for backwards compatibility",
			npmVersion:     "8.3.1",
			nodeVersion:    "10.24.1",
			want:           "install",
			targetPlatform: env.TargetPlatformFunctions,
		},
		{
			name:       "5.7.0 should return install",
			npmVersion: "5.7.0",
			want:       "install",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) (string, error) { return tc.npmVersion, nil }
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { nodeVersion = fn }(nodeVersion)
			nodeVersion = func(*gcpbuildpack.Context) (string, error) { return tc.nodeVersion, nil }
			if tc.targetPlatform != "" {
				t.Setenv(env.XGoogleTargetPlatform, tc.targetPlatform)
			}

			got, err := NPMInstallCommand(nil)
			if err != nil {
				t.Fatalf("npm %v: NPMInstallCommand(nil) got error: %v", tc.npmVersion, err)
			}
			if got != tc.want {
				t.Errorf("npm %v: NPMInstallCommand(nil) = %q, want %q", tc.npmVersion, got, tc.want)
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
			defer func(fn func(*gcpbuildpack.Context) (string, error)) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) (string, error) { return tc.version, nil }

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

func TestDetermineBuildCommands(t *testing.T) {
	testsCases := []struct {
		name               string
		pjs                string
		nodeRunScriptSet   bool
		nodeRunScriptValue string // ignored if `nodeRunScriptSet == false`
		targetPlatformSet  bool
		want               []string
		wantIsCustomBuild  bool
	}{
		{
			name:             "no build",
			pjs:              "",
			nodeRunScriptSet: false,
			want:             []string{},
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "lint,clean,build",
			want:               []string{"npm run lint", "npm run clean", "npm run build"},
			wantIsCustomBuild:  true,
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS single value",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "lint",
			want:               []string{"npm run lint"},
			wantIsCustomBuild:  true,
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS trim whitespace",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "    	lint	,	 build",
			want:               []string{"npm run lint", "npm run build"},
			wantIsCustomBuild:  true,
		},
		{
			name: "build script",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			targetPlatformSet: true,
			want:              []string{"npm run build"},
		},
		{
			name: "build script runs regardless of target platform",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			targetPlatformSet: false,
			want:              []string{"npm run build"},
		},
		{
			name: "gcp-build script",
			pjs: `{
				"scripts": {
					"gcp-build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			want:              []string{"npm run gcp-build"},
			wantIsCustomBuild: true,
		},
		{
			name: "GOOGLE_NODE_RUN_SCRIPTS highest precedence",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "from-env",
			want:               []string{"npm run from-env"},
			wantIsCustomBuild:  true,
		},
		{
			name: "gcp-build higher precedence than build",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			want:              []string{"npm run gcp-build"},
			wantIsCustomBuild: true,
		},
		{
			name: "empty gcp-build higher precedence than build and runs nothing",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "  "
				}
			}`,
			want:              []string{},
			wantIsCustomBuild: true,
		},
		{
			name: "empty build runs nothing",
			pjs: `{
				"scripts": {
					"build": ""
				}
			}`,
			want:              []string{},
			wantIsCustomBuild: false,
		},
		{
			name: "setting empty GOOGLE_NODE_RUN_SCRIPTS runs nothing",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "",
			want:               []string{},
			wantIsCustomBuild:  true,
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nodeRunScriptSet {
				t.Setenv(GoogleNodeRunScriptsEnv, tc.nodeRunScriptValue)
			}

			if tc.targetPlatformSet {
				t.Setenv(env.XGoogleTargetPlatform, env.TargetPlatformAppEngine)
			}

			var pjs *PackageJSON
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}
			got, isCustomBuild := DetermineBuildCommands(pjs, "npm")
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("DetermineBuildCommands() mismatch (-want +got):\n%s", diff)
			}

			if isCustomBuild != tc.wantIsCustomBuild {
				t.Errorf("DetermineBuildCommands() is custom build mismatch, got: %t, want: %t", isCustomBuild, tc.wantIsCustomBuild)
			}
		})
	}
}
