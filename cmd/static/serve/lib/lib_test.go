// Copyright 2026 Google LLC
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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  int
	}{
		{
			name: "with_index_html_at_root",
			files: map[string]string{
				"index.html": "hello",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "with_dist_directory",
			files: map[string]string{
				"dist/index.html": "hello",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: 0,
		},
		{
			name: "with_index_html_but_release_track_not_alpha",
			files: map[string]string{
				"index.html": "hello",
			},
			want: 100,
		},
		{
			name:  "neither_index_nor_build_folders",
			files: map[string]string{},
			envs:  []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want:  100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  string
	}{
		{
			name: "with_dist_directory_priority",
			files: map[string]string{
				"dist/index.html": "hello",
				"index.html":      "root dummy file",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			want: "Target static asset folder found: dist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildpacktest.RunBuild(t, BuildFn, buildpacktest.WithFiles(tc.files), buildpacktest.WithEnvs(tc.envs...), buildpacktest.WithTestName(tc.name))
			if err != nil {
				t.Fatalf("error running build: %v, result: %#v", err, result)
			}
			if !strings.Contains(result.Output, tc.want) {
				t.Errorf("RunBuild().Output = %q, want %q", result.Output, tc.want)
			}
		})
	}
}
