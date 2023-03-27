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

package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "with package",
			files: map[string]string{
				"index.js": "",

				"package.json": "",
			},
			want: 0,
		},
		{
			name: "without package",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name              string
		app               string
		envs              []string
		opts              []bpt.Option
		mocks             []*mockprocess.Mock
		wantExitCode      int // 0 if unspecified
		wantCommands      []string
		doNotWantCommands []string
	}{
		{
			name: "package-lock.json",
			app:  "package_lock",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=production",
			},
		},
		{
			name: "respect user NODE_ENV",
			app:  "package_lock",
			envs: []string{"NODE_ENV=custom"},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=custom",
			},
		},
		{
			name: "gcp-build script",
			app:  "gcp_build_npm",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=development",
				"npm run gcp-build",
			},
		},
		{
			name: "build script",
			app:  "typescript",
			envs: []string{"GOOGLE_EXPERIMENTAL_NODEJS_NPM_BUILD_ENABLED=true"},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=development",
				"npm run build",
			},
		},
		{
			name: "node scripts env set",
			app:  "gcp_build_npm",
			envs: []string{fmt.Sprintf("%s=lint", googleNodeRunScriptsEnv)},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=development",
				"npm run lint",
			},
			doNotWantCommands: []string{
				"npm run build",
				"npm run gcp-build",
			},
		},
		{
			name: "node scripts env set but empty",
			app:  "gcp_build_npm",
			envs: []string{fmt.Sprintf("%s=", googleNodeRunScriptsEnv)},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm install.*NODE_ENV=production",
			},
			doNotWantCommands: []string{
				"npm run build",
				"npm run gcp-build",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithApp(tc.app),
				bpt.WithEnvs(tc.envs...),
				bpt.WithExecMocks(tc.mocks...),
			}
			opts = append(opts, tc.opts...)
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
			}

			for _, cmd := range tc.doNotWantCommands {
				if result.CommandExecuted(cmd) {
					t.Errorf("expected command %q not to be executed, but it was, build output: %s", cmd, result.Output)
				}
			}
		})
	}
}

func TestDetermineBuildCommands(t *testing.T) {
	testsCases := []struct {
		name                       string
		pjs                        string
		nodeRunScriptSet           bool
		nodeRunScriptValue         string // ignored if `nodeRunScriptSet == false`
		nodejsNPMBuildExperimentOn bool
		want                       []string
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
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS single value",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "lint",
			want:               []string{"npm run lint"},
		},
		{
			name:               "GOOGLE_NODE_RUN_SCRIPTS trim whitespace",
			nodeRunScriptSet:   true,
			nodeRunScriptValue: "    	lint	,	 build",
			want:               []string{"npm run lint", "npm run build"},
		},
		{
			name: "build script",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			nodejsNPMBuildExperimentOn: true,
			want:                       []string{"npm run build"},
		},
		{
			name: "build script experiment off",
			pjs: `{
				"scripts": {
					"build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			nodejsNPMBuildExperimentOn: false,
			want:                       []string{},
		},
		{
			name: "gcp-build script",
			pjs: `{
				"scripts": {
					"gcp-build": "tsc --build",
					"clean": "tsc --build --clean"
				}
			}`,
			want: []string{"npm run gcp-build"},
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
		},
		{
			name: "gcp-build higher precedence than build",
			pjs: `{
				"scripts": {
					"build": "tsc --build --clean",
					"gcp-build": "tsc --build"
				}
			}`,
			want: []string{"npm run gcp-build"},
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
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nodeRunScriptSet {
				t.Setenv(googleNodeRunScriptsEnv, tc.nodeRunScriptValue)
			}

			if tc.nodejsNPMBuildExperimentOn {
				t.Setenv(nodejsNPMBuildEnv, "true")
			}

			var pjs *nodejs.PackageJSON
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}
			got := determineBuildCommands(pjs)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("determineBuildCommands() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
