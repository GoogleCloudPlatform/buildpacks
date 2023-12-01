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
	"fmt"
	"testing"

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
		files             map[string]string
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
			envs: []string{fmt.Sprintf("%s=lint", nodejs.GoogleNodeRunScriptsEnv)},
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
			envs: []string{fmt.Sprintf("%s=", nodejs.GoogleNodeRunScriptsEnv)},
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
		{
			name: "node rebuild for vendored deps",
			envs: []string{"GOOGLE_VENDOR_NPM_DEPENDENCIES=true"},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^npm --version$`, mockprocess.WithStdout("0.0.0")),
			},
			wantCommands: []string{
				"npm rebuild",
			},
			doNotWantCommands: []string{
				"npm ci",
			},
			files: map[string]string{
				"node_modules/index.js": "",
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
				bpt.WithFiles(tc.files),
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
