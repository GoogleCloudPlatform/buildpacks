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
			name: "pom.xml",
			files: map[string]string{
				"pom.xml": "",
			},
			env:  []string{"GOOGLE_CLEAR_SOURCE=true"},
			want: 0,
		},
		{
			name: "build.gradle",
			files: map[string]string{
				"build.gradle": "",
			},
			env:  []string{"GOOGLE_CLEAR_SOURCE=true"},
			want: 0,
		},
		{
			name: "build.gradle.kts",
			files: map[string]string{
				"build.gradle.kts": "",
			},
			env:  []string{"GOOGLE_CLEAR_SOURCE=true"},
			want: 0,
		},
		{
			name:  "none of pom.xml, build.gradle, build.gradle.kts",
			files: map[string]string{},
			env:   []string{"GOOGLE_CLEAR_SOURCE=true"},
			want:  100,
		},
		{
			name: "pom.xml exists but env var not set",
			files: map[string]string{
				"pom.xml": "",
			},
			want: 100,
		},
		{
			name: "build.gradle exists but env var not set",
			files: map[string]string{
				"build.gradle": "",
			},
			want: 100,
		},
		{
			name: "build.gradle.kts exists but env var not set",
			files: map[string]string{
				"build.gradle.kts": "",
			},
			want: 100,
		},
		{
			name:  "none of pom.xml, build.gradle, build.gradle.kts exist and env var not set",
			files: map[string]string{},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
