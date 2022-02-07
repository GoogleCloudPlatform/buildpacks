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
		name   string
		files  map[string]string
		want   int
		envVal string
	}{
		{
			name: "composer.json file",
			files: map[string]string{
				"composer.json": "",
			},
			want: 0,
		},
		{
			name: "php files",
			files: map[string]string{
				"index.php": "",
			},
			want: 0,
		},
		{
			name: "composer.json and php file",
			files: map[string]string{
				"composer.json": "",
				"index.php":     "",
			},
			want: 0,
		},
		{
			name:  "no composer.json and no php files",
			files: map[string]string{},
			want:  100,
		},
		{
			name:  "no env",
			files: map[string]string{},
			want:  100,
		},
		{
			name:   "env is disabled",
			files:  map[string]string{},
			want:   100,
			envVal: "False",
		},
		{
			name:   "env is invalid",
			files:  map[string]string{},
			want:   100,
			envVal: "Flase",
		},
	}
	for _, tc := range testCases {
		if tc.envVal == "" {
			tc.envVal = "True"
		}
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, []string{"GOOGLE_USE_EXPERIMENTAL_PHP_RUNTIME=" + tc.envVal}, tc.want)
		})
	}
}
