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
			name: "py files",
			files: map[string]string{
				"main.py": "",
			},
			want: 0,
		},
		{
			name: "py files and runtime set to python",
			files: map[string]string{
				"main.py": "",
			},
			env:  []string{"GOOGLE_RUNTIME=python"},
			want: 0,
		},
		{
			name: "py files and runtime set to php",
			files: map[string]string{
				"main.py": "",
			},
			env:  []string{"GOOGLE_RUNTIME=php"},
			want: 100,
		},
		{
			name:  "no py files",
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

func TestParseExecPrefix(t *testing.T) {
	testCases := []struct {
		sysConfig string
		want      string
		wantErr   bool
	}{
		{
			sysConfig: "",
			want:      "",
			wantErr:   true,
		},
		{
			sysConfig: `installed_base = "/layers/google.python.runtime/python"`,
			want:      "",
			wantErr:   true,
		},
		{
			sysConfig: `
exec_prefix = "/opt/python3.11"
installed_base = "/layers/google.python.runtime/python"
			`,
			want: "/opt/python3.11",
		},
		{
			sysConfig: `
exec_prefix = "/opt/python3.9"
installed_base = "/layers/google.python.runtime/python"
			`,
			want: "/opt/python3.9",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.sysConfig, func(t *testing.T) {
			got, err := parseExecPrefix(tc.sysConfig)
			if (err == nil) == tc.wantErr {
				t.Errorf("parseExecPrefix() got err: %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("parseExecPrefix(%q) = %q, want %q", tc.sysConfig, got, tc.want)
			}
		})
	}
}
