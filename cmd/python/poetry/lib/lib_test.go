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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		envs  []string
		want  int
		files map[string]string
	}{
		{
			name: "should_opt_in_for_poetry_in_beta",
			envs: []string{env.ReleaseTrack + "=BETA"},
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]`,
			},
			want: 0,
		},
		{
			name: "should_opt_in_for_poetry_in_alpha",
			envs: []string{env.ReleaseTrack + "=ALPHA"},
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]`,
			},
			want: 0,
		},
		{
			name: "should_opt_in_for_poetry_lock_in_alpha",
			envs: []string{env.ReleaseTrack + "=ALPHA"},
			files: map[string]string{
				"pyproject.toml": ``,
				"poetry.lock":    ``,
			},
			want: 0,
		},
		{
			name: "should_opt_out_for_poetry_in_ga",
			envs: []string{env.ReleaseTrack + "=GA"},
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]`,
			},
			want: 100,
		},
		{
			name: "should_opt_out_for_poetry_with_no_release_track",
			files: map[string]string{
				"pyproject.toml": `[tool.poetry]`,
			},
			want: 100,
		},
		{
			name: "should_opt_out_when_no_pyproject.toml",
			envs: []string{env.ReleaseTrack + "=ALPHA"},
			files: map[string]string{
				"main.py": "",
			},
			want: 100,
		},
		{
			name: "should_opt_out_when_pyproject.toml_has_no_poetry_section",
			envs: []string{env.ReleaseTrack + "=ALPHA"},
			files: map[string]string{
				"pyproject.toml": "[tool.other]",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}
