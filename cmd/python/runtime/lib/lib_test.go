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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb/v2"
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
	testDetectAutodetectPlan(t)
}

// testDetectAutodetectPlan runs subtests of TestDetect, which buildpacktest requires for detect
// tests, for the build plan that records the runtime for automatic runtime detection.
func testDetectAutodetectPlan(t *testing.T) {
	const versions = `X_GOOGLE_RUNTIME_VERSIONS={"nodejs24":"24.22.1","python314":"3.14.10"}`
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  []libcnb.BuildPlan
	}{
		{
			name:  "autodetect_python_version_file",
			files: map[string]string{"main.py": "", ".python-version": "3.14"},
			env:   []string{versions},
			want:  autodetectPlans("python314", "3.14.10"),
		},
		{
			name:  "autodetect_unpinned",
			files: map[string]string{"main.py": ""},
			env:   []string{versions},
			want:  autodetectPlans("python314", "3.14.10"),
		},
		{
			name:  "autodetect_google_python_version_env",
			files: map[string]string{"main.py": ""},
			env:   []string{versions, "GOOGLE_PYTHON_VERSION=3.14.10"},
			want:  autodetectPlans("python314", "3.14.10"),
		},
		{
			name:  "autodetect_google_runtime_override",
			files: map[string]string{"main.py": ""},
			env:   []string{versions, "GOOGLE_RUNTIME=python"},
			want:  autodetectPlans("python314", "3.14.10"),
		},
		{
			name:  "autodetect_version_not_eligible",
			files: map[string]string{"main.py": "", ".python-version": "3.12"},
			env:   []string{versions},
		},
		{
			name:  "autodetect_pinned_version_uses_release_line",
			files: map[string]string{"main.py": "", ".python-version": "3.14.9"},
			env:   []string{versions},
			want:  autodetectPlans("python314", "3.14.10"),
		},
		{
			name:  "autodetect_empty_python_version_file",
			files: map[string]string{"main.py": "", ".python-version": ""},
			env:   []string{versions},
		},
		{
			name:  "autodetect_not_enabled",
			files: map[string]string{"main.py": "", ".python-version": "3.14"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithPlans(t, DetectFn, tc.name, tc.files, tc.env, 0, tc.want)
		})
	}
}

func autodetectPlans(name, version string) []libcnb.BuildPlan {
	return []libcnb.BuildPlan{runtime.AutodetectPlan(&runtime.AutodetectedRuntime{Name: name, Version: version})}
}
