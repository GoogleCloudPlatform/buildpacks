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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb/v2"
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
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: 0,
		},
		{
			name: "with_index_html_only_in_dist_directory",
			files: map[string]string{
				"dist/index.html": "hello",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: 100,
		},
		{
			name: "with_index_html_only_in_build_directory",
			files: map[string]string{
				"build/index.html": "hello",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: 100,
		},
		{
			name: "with_index_html_directory_at_root",
			files: map[string]string{
				"index.html/placeholder": "not a file",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: 100,
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
			envs:  []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want:  100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
	testDetectAutodetectPlan(t)
}

// testDetectAutodetectPlan runs subtests of TestDetect, which buildpacktest requires for detect
// tests, for the build plan that records the runtime for automatic runtime detection.
func testDetectAutodetectPlan(t *testing.T) {
	files := map[string]string{"index.html": "hello"}
	testCases := []struct {
		name string
		envs []string
		want []libcnb.BuildPlan
	}{
		{
			name: "autodetect_static24",
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA", `X_GOOGLE_RUNTIME_VERSIONS={"nodejs24":"24.22.1","static24":"24"}`},
			want: []libcnb.BuildPlan{runtime.AutodetectPlan(&runtime.AutodetectedRuntime{Name: "static24", Version: "24"})},
		},
		{
			name: "autodetect_static_not_eligible",
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA", `X_GOOGLE_RUNTIME_VERSIONS={"nodejs24":"24.22.1"}`},
		},
		{
			name: "autodetect_not_enabled",
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithPlans(t, DetectFn, tc.name, files, tc.envs, 0, tc.want)
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
			name: "serves_application_root",
			files: map[string]string{
				"index.html": "hello",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: "Serving static files from the application root",
		},
		{
			name: "serves_application_root_even_with_dist_directory",
			files: map[string]string{
				"dist/index.html": "compiled output",
				"index.html":      "root file",
			},
			envs: []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			want: "Serving static files from the application root",
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
