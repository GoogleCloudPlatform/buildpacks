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

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
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
			name: "with target and nodejs8",
			env:  []string{"GOOGLE_FUNCTION_TARGET=helloWorld", "GOOGLE_RUNTIME=nodejs8"},
			want: 0,
		},
		{
			name: "with target, but without nodejs8",
			env:  []string{"GOOGLE_FUNCTION_TARGET=helloWorld", "GOOGLE_RUNTIME=nodejs10"},
			want: 100,
		},
		{
			name: "with target, but without GOOGLE_RUNTIME",
			env:  []string{"GOOGLE_FUNCTION_TARGET=helloWorld"},
			want: 100,
		},
		{
			name: "without target, but with nodejs8",
			env:  []string{"GOOGLE_RUNTIME=nodejs8"},
			want: 100,
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
