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

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		stack string
		want  int
	}{
		{
			name: "with target",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			want: 0,
		},
		{
			name: "without target",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name         string
		app          string
		envs         []string
		fnPkgName    string
		opts         []buildpacktest.Option
		mocks        []*mockprocess.Mock
		wantExitCode int // 0 if unspecified
		wantCommands []string
	}{
		{
			name:      "go mod function with framework",
			app:       "with_framework",
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
			},
			wantCommands: []string{fmt.Sprintf("go mod tidy")},
		},
		{
			name: "go mod function with framework without injection",
			app:  "with_framework",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
				mockprocess.New(`^go list -m -f {{.Version}}.*`, mockprocess.WithStdout("v1.0.0")),
			},
			wantCommands: []string{fmt.Sprintf("go mod tidy")},
		},
		{
			name:      "go mod function without framework",
			app:       "no_framework",
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
			},
			wantCommands: []string{
				fmt.Sprintf("go mod edit -require %s", functionsFrameworkModule),
				"go mod tidy",
			},
		},
		{
			name: "go mod function without framework without injection",
			app:  "no_framework",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
			},
			wantExitCode: 1,
		},
		{
			name:         "vendored function",
			app:          "no_framework_vendored_no_go_mod",
			fnPkgName:    "myfunc",
			wantCommands: []string{"go mod vendor"},
		},
		{
			name: "vendored function without injection",
			app:  "no_framework_vendored_no_go_mod",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
			fnPkgName:    "myfunc",
			wantExitCode: 1,
		},
		{
			name: "with framework vendored for go 1.22 and below",
			app:  "with_framework_vendored",
			envs: []string{
				"GOOGLE_RUNTIME_VERSION=1.22.11",
			},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
				mockprocess.New(`^go list -m -f {{.Version}}.*`, mockprocess.WithStdout("v1.0.0")),
				mockprocess.New(`^go version$`, mockprocess.WithStdout("go version go1.22 linux/amd64")),
			},
		},
		{
			name: "with framework vendored for go 1.23 and above",
			app:  "with_framework_vendored",
			envs: []string{
				"GOOGLE_RUNTIME_VERSION=1.23.5",
			},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
				mockprocess.New(`^go list -m -f {{.Version}}.*`, mockprocess.WithStdout("v1.0.0")),
			},
		},
		{
			name:      "with framework vendored for go 1.23 and above, version not specified",
			app:       "with_framework_vendored",
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
				mockprocess.New(`^go list -m -f {{.Version}}.*`, mockprocess.WithStdout("v1.0.0")),
			},
		},
		{
			name: "with framework vendored without injection",
			app:  "with_framework_vendored",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
				mockprocess.New(`^go list -m -f {{.Version}}.*`, mockprocess.WithStdout("v1.0.0")),
			},
		},
		{
			name:         "without framework vendored",
			app:          "without_framework_vendored",
			fnPkgName:    "myfunc",
			mocks:        []*mockprocess.Mock{},
			wantExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envs := []string{
				"GOOGLE_FUNCTION_TARGET=Func",
			}
			envs = append(envs, tc.envs...)
			mocks := []*mockprocess.Mock{
				mockprocess.New("get_package", mockprocess.WithStdout(fmt.Sprintf(`{"name":"%s"}`, tc.fnPkgName))),
			}
			mocks = append(mocks, tc.mocks...)

			opts := []buildpacktest.Option{
				buildpacktest.WithTestName(tc.name),
				buildpacktest.WithApp(tc.app),
				buildpacktest.WithEnvs(envs...),
				buildpacktest.WithExecMocks(mocks...),
			}
			opts = append(opts, tc.opts...)
			result, err := buildpacktest.RunBuild(t, buildFn, opts...)
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
		})
	}
}
