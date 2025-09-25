// Copyright 2025 Google LLC
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

package lib

import (
	"fmt"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "build.gradle",
			files: map[string]string{
				"build.gradle": "",
			},
			want: 0,
		},
		{
			name: "build.gradle.kts",
			files: map[string]string{
				"build.gradle.kts": "",
			},
			want: 0,
		},
		{
			name:  "no files",
			files: map[string]string{},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		name              string
		app               string
		envs              []string
		opts              []buildpacktest.Option
		mocks             []*mockprocess.Mock
		wantCommands      []string
		doNotWantCommands []string
		files             map[string]string
	}{
		{
			name: "maven build argument",
			app:  "gradle_micronaut",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bash -c command -v gradle || true`, mockprocess.WithStdout("Gradle 0.0.0")),
			},
			envs: []string{fmt.Sprintf("%s=clean assemble", java.GradleBuildArgs)},
			wantCommands: []string{
				"gradle clean assemble",
			},
			doNotWantCommands: []string{
				"gradle clean assemble -x test --build-cache",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []buildpacktest.Option{
				buildpacktest.WithTestName(tc.name),
				buildpacktest.WithApp(tc.app),
				buildpacktest.WithEnvs(tc.envs...),
				buildpacktest.WithExecMocks(tc.mocks...),
			}

			opts = append(opts, tc.opts...)
			result, err := buildpacktest.RunBuild(t, BuildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
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
