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
		want  int
	}{
		{
			name: "with package",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			want: 0,
		},
		{
			name: "with package and runtime set to nodejs",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=nodejs"},
			want: 0,
		},
		{
			name: "with FIREBASE_OUTPUT_BUNDLE_DIR env variable",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=nodejs", "FIREBASE_OUTPUT_BUNDLE_DIR=/output/dir"},
			want: 0,
		},
		{
			name: "with package and runtime set to python",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=python"},
			want: 100,
		},
		{
			name: "without package",
			files: map[string]string{
				"index.js": "",
			},
			want: 0,
		},
		{
			name: "without js files",
			files: map[string]string{
				"index.txt": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
