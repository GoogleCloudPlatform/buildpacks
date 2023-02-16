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
			bpt.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name         string
		app          string
		envs         []string
		fnPkgName    string
		opts         []bpt.Option
		mocks        []*mockprocess.Mock
		wantExitCode int // 0 if unspecified
		wantCommands []string
	}{
		{
			name:      "go mod function with framework",
			app:       "with_framework",
			envs:      []string{"GOOGLE_FUNCTION_TARGET=Func"},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc")),
			},
			wantCommands: []string{fmt.Sprintf("go mod tidy")},
		},
		{
			name:      "go mod function without framework",
			app:       "no_framework",
			envs:      []string{"GOOGLE_FUNCTION_TARGET=Func"},
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
			name:         "vendored function",
			app:          "no_framework_vendored",
			envs:         []string{"GOOGLE_FUNCTION_TARGET=Func"},
			fnPkgName:    "myfunc",
			wantCommands: []string{"go mod vendor"},
		},
		{
			name:      "go mod with version",
			app:       "with_versioned_mod",
			envs:      []string{"GOOGLE_FUNCTION_TARGET=Func"},
			fnPkgName: "myfunc",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^go list -m$`, mockprocess.WithStdout("example.com/myfunc/v3")),
			},
			wantCommands: []string{
				"go mod edit -require example.com/myfunc/v3@v3",
				"go mod edit -replace example.com/myfunc/v3=",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mocks := []*mockprocess.Mock{
				mockprocess.New("get_package", mockprocess.WithStdout(fmt.Sprintf(`{"name":"%s"}`, tc.fnPkgName))),
			}
			mocks = append(mocks, tc.mocks...)

			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithApp(tc.app),
				bpt.WithEnvs(tc.envs...),
				bpt.WithExecMocks(mocks...),
			}
			opts = append(opts, tc.opts...)
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v,logs: %s", err, result.Output)
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

func TestParseModuleVersion(t *testing.T) {
	testCases := []struct {
		module string
		want   string
	}{
		{
			module: "example.com/v2",
			want:   "v2",
		},
		{
			module: "/v123",
			want:   "v123",
		},
		{
			module: "/v",
			want:   "",
		},
		{
			module: "example.com/no/major/version",
			want:   "",
		},
	}

	for _, tc := range testCases {
		got, err := parseModuleVersion(tc.module)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != tc.want {
			t.Errorf("parsed version mismatch, got: %q, want: %q", got, tc.want)
		}
	}
}
