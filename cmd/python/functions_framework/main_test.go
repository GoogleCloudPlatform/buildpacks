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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
)

func TestContainsFF(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "ff_present",
			str:  "functions-framework==19.9.0\nflask\n",
			want: true,
		},
		{
			name: "ff_present_with_comment",
			str:  "functions-framework #my-comment\nflask\n",
			want: true,
		},
		{
			name: "ff_present_second_line",
			str:  "flask\nfunctions-framework==19.9.0",
			want: true,
		},
		{
			name: "no_ff_present",
			str:  "functions-framework-example==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "ff_egg_present",
			str:  "git+git://github.com/functions-framework@master#egg=functions-framework\nflask\n",
			want: true,
		},
		{
			name: "ff_egg_not_present",
			str:  "git+git://github.com/functions-framework-example@master#egg=functions-framework-example\nflask\n",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsFF(tc.str)
			if got != tc.want {
				t.Errorf("containsFF() got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name         string
		app          string
		envs         []string
		opts         []buildpacktest.Option
		mocks        []*mockprocess.Mock
		wantExitCode int // 0 if unspecified
		wantCommands []string
	}{
		{
			name: "with framework",
			app:  "with_framework",
		},
		{
			name: "with framework without injection",
			app:  "with_framework",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
		},
		{
			name: "without framework",
			app:  "without_framework",
		},
		{
			name: "without framework without injection",
			app:  "without_framework",
			envs: []string{
				"GOOGLE_SKIP_FRAMEWORK_INJECTION=True",
			},
			wantExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envs := []string{
				"GOOGLE_FUNCTION_TARGET=testFunction",
			}
			envs = append(envs, tc.envs...)
			mocks := []*mockprocess.Mock{
				mockprocess.New(`^python3 -m compileall -f -q .$`),
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
			env:  []string{"GOOGLE_FUNCTION_TARGET=helloWorld"},
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
