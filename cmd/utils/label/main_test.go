// Copyright 2022 Google LLC
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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

const labelLog = "Adding image label"

func TestDetect(t *testing.T) {
	buildpacktest.TestDetect(t, DetectFn, "Always opt-in", map[string]string{}, []string{}, 0)
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name string
		app  string
		envs []string
		want string
	}{
		{
			name: "valid label env var",
			app:  "with_framework",
			envs: []string{"GOOGLE_LABEL_FOO=bar"},
			want: labelLog + " google.foo: bar",
		},
		{
			name: "random env var",
			app:  "with_framework",
			envs: []string{"GOOGLE_FOO=bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildpacktest.RunBuild(t, BuildFn, buildpacktest.WithEnvs(tc.envs...), buildpacktest.WithTestName(tc.name))
			if err != nil {
				t.Fatalf("error running build: %v, result: %#v", err, result)
			}
			if tc.want == "" && strings.Contains(result.Output, labelLog) {
				t.Errorf("RunBuild().Output = %q, want without %q", result.Output, labelLog)
			}
			if tc.want != "" && !strings.Contains(result.Output, tc.want) {
				t.Errorf("RunBuild().Output = %q, want %q ", result.Output, tc.want)
			}
		})
	}
}
