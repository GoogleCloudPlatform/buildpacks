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

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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
			name: "GraalVM env var set to True",
			env:  []string{"GOOGLE_JAVA_USE_NATIVE_IMAGE=True"},
			want: 0,
		},
		{
			name: "GraalVM env var set to true (lower-cased)",
			env:  []string{"GOOGLE_JAVA_USE_NATIVE_IMAGE=true"},
			want: 0,
		},
		{
			name: "GraalVM env var set to true (number)",
			env:  []string{"GOOGLE_JAVA_USE_NATIVE_IMAGE=1"},
			want: 0,
		},
		{
			name: "GraalVM env var set to False",
			env:  []string{"GOOGLE_JAVA_USE_NATIVE_IMAGE=False"},
			want: 100,
		},
		{
			name: "Without env var",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}
