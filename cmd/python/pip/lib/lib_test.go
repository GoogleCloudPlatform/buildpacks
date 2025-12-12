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
			name: "requirements file",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "",
			},
			want: 0,
		},
		{
			// Opt-in with no requirements in case there's a build plan.
			name: "no requirements",
			files: map[string]string{
				"main.py": "",
			},
			want: 0,
		},
		{
			name: "pyproject.toml_file_when_env_var_is_pip_in_beta",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-pip-project"`,
			},
			env:  []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip", "X_GOOGLE_RELEASE_TRACK=BETA"},
			want: 0,
		},
		{
			name: "pyproject.toml_file_when_env_var_is_pip_in_ga_for_python_313",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "my-pip-project"`,
			},
			env:  []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
