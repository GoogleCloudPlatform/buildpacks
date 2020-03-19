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
		want  int
	}{
		{
			name: "with firebase with target",
			files: map[string]string{
				"index.js": "",
				"package.json": `
{
  "dependencies": {
    "firebase-functions": "^3.4.0"
  }
}
`,
			},
			env:  []string{"FUNCTION_TARGET=helloWorld"},
			want: 0,
		},
		{
			name: "with firebase without target",
			files: map[string]string{
				"index.js": "",
				"package.json": `
{
  "dependencies": {
    "firebase-functions": "^3.4.0"
  }
}
`,
			},
			env:  []string{"FOO=helloWorld"},
			want: 100,
		},
		{
			name: "without firebase with target",
			files: map[string]string{
				"index.js":     "",
				"package.json": "{}",
			},
			env:  []string{"FUNCTION_TARGET=helloWorld"},
			want: 100,
		},
		{
			name: "without firebase without target",
			files: map[string]string{
				"index.js": "",
			},
			env:  []string{"FOO=helloWorld"},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
