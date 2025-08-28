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
		env   []string
		files map[string]string
		want  int
	}{
		{
			name:  "no env var, no app.yaml",
			env:   []string{""},
			files: map[string]string{},
			want:  100,
		},
		{
			name:  "has env var, no app.yaml",
			env:   []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{},
			want:  100,
		},
		{
			name: "has env var, mismatch yaml file",
			env:  []string{"GAE_APPLICATION_YAML_PATH=foo.yaml"},
			files: map[string]string{
				"app.yaml": "env: flex",
			},
			want: 100,
		},
		{
			name:  "has env var, no app.yaml",
			env:   []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{},
			want:  100,
		},
		{
			name: "empty app.yaml",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "",
			},
			want: 100,
		},
		{
			name: "app.yaml without env: flex",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "env: other",
			},
			want: 100,
		},
		{
			name: "app.yaml env: flex",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "env: flex",
			},
			want: 0,
		},
		{
			name: "app.yaml env: flexible",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "env: flexible",
			},
			want: 0,
		},
		{
			name: "app.yaml env: flex with whitespaces",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "  env  :  flex  ",
			},
			want: 0,
		},
		{
			name: "app.yaml env: flex with newline",
			env:  []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			files: map[string]string{
				"app.yaml": "runtime: python\n  env  :  flex  \n",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
