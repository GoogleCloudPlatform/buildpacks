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

package lib

import (
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "without package without pnpm",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
		{
			name: "with package without pnpm",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			want: 100,
		},
		{
			name: "without package with pnpm",
			files: map[string]string{
				"index.js":       "",
				"pnpm-lock.yaml": "",
			},
			want: 100,
		},
		{
			name: "with pnpm and package",
			files: map[string]string{
				"index.js":       "",
				"pnpm-lock.yaml": "",
				"package.json":   "",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}
