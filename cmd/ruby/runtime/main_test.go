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
			name: "Gemfile file",
			files: map[string]string{
				"Gemfile": "",
			},
			want: 0,
		},
		{
			name: "gems.rb file",
			files: map[string]string{
				"gems.rb": "",
			},
			want: 0,
		},
		{
			name: "rb files",
			files: map[string]string{
				"main.rb": "",
			},
			want: 0,
		},
		{
			name: "rb files and runtime set to ruby",
			files: map[string]string{
				"main.rb": "",
			},
			env:  []string{"GOOGLE_RUNTIME=ruby"},
			want: 0,
		},
		{
			name: "rb files and runtime set to php",
			files: map[string]string{
				"main.rb": "",
			},
			env:  []string{"GOOGLE_RUNTIME=php"},
			want: 100,
		},
		{
			name:  "no ruby files",
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
