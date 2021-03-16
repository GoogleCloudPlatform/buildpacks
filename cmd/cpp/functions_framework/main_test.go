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
			name: "with target and files [cc]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cc": "",
			},
			want: 0,
		},
		{
			name: "with target and files [cpp]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cpp": "",
			},
			want: 0,
		},
		{
			name: "with target and files [cxx]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cxx": "",
			},
			want: 0,
		},
		{
			name: "with target and CMake",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"CMakeLists.txt": "",
			},
			want: 0,
		},
		{
			name: "with target no files",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			want: 100,
		},
		{
			name: "without target but with files",
			files: map[string]string{
				"main.cc": "",
			},
			want: 100,
		},
		{
			name: "without target or files",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}
