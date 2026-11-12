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
			name: "with package",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			want: 0,
		},
		{
			name: "with package and runtime set to nodejs",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=nodejs"},
			want: 0,
		},
		{
			name: "with FIREBASE_OUTPUT_BUNDLE_DIR env variable",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=nodejs", "FIREBASE_OUTPUT_BUNDLE_DIR=/output/dir"},
			want: 0,
		},
		{
			name: "with package and runtime set to python",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			env:  []string{"GOOGLE_RUNTIME=python"},
			want: 100,
		},
		{
			name: "without package",
			files: map[string]string{
				"index.js": "",
			},
			want: 0,
		},
		{
			name: "with_ts_files_and_node_version_>=_24",
			files: map[string]string{
				"index.ts": "",
			},
			env:  []string{"GOOGLE_RUNTIME_VERSION=24.1.0"},
			want: 0,
		},
		{
			name: "with_ts_files_and_node_version_<_24",
			files: map[string]string{
				"index.ts": "",
			},
			env:  []string{"GOOGLE_RUNTIME_VERSION=23.1.0"},
			want: 100,
		},
		{
			name: "without js files",
			files: map[string]string{
				"index.txt": "",
			},
			want: 100,
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
			name:  "autodetect_engines_24",
			files: map[string]string{"package.json": `{"engines": {"node": "24.x"}}`},
			env:   []string{versions},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_open_range",
			files: map[string]string{"package.json": `{"engines": {"node": ">=16.0.0"}}`},
			env:   []string{versions},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_unpinned",
			files: map[string]string{"package.json": `{}`},
			env:   []string{versions},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_js_files_only",
			files: map[string]string{"index.js": ""},
			env:   []string{versions},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_runtime_version_env",
			files: map[string]string{"package.json": `{"engines": {"node": "22.x"}}`},
			env:   []string{versions, "GOOGLE_RUNTIME_VERSION=24.22.1"},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_google_runtime_override",
			files: map[string]string{"index.js": ""},
			env:   []string{versions, "GOOGLE_RUNTIME=nodejs"},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_pinned_version_uses_release_line",
			files: map[string]string{"package.json": `{"engines": {"node": "24.21.0"}}`},
			env:   []string{versions},
			want:  autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_version_not_eligible",
			files: map[string]string{"package.json": `{"engines": {"node": "22.x"}}`},
			env:   []string{versions},
		},
		{
			name:  "autodetect_invalid_package_json",
			files: map[string]string{"package.json": `{`},
			env:   []string{versions},
		},
		{
			name: "autodetect_rails_app",
			files: map[string]string{
				"bin/rails":    "",
				"package.json": `{"engines": {"node": "24.x"}}`,
			},
			env: []string{versions},
		},
		{
			name: "autodetect_ruby_app_with_gemfile",
			files: map[string]string{
				"Gemfile":      "",
				"package.json": `{"engines": {"node": "24.x"}}`,
			},
			env: []string{versions},
		},
		{
			name: "autodetect_ruby_app_with_gems_rb",
			files: map[string]string{
				"gems.rb":      "",
				"package.json": `{"engines": {"node": "24.x"}}`,
			},
			env: []string{versions},
		},
		{
			name: "autodetect_ruby_app_with_rb_files",
			files: map[string]string{
				"lib/app.rb":   "",
				"package.json": `{"engines": {"node": "24.x"}}`,
			},
			env: []string{versions},
		},
		{
			name: "autodetect_rb_file_in_node_modules",
			files: map[string]string{
				"node_modules/some-pkg/tool.rb": "",
				"package.json":                  `{"engines": {"node": "24.x"}}`,
			},
			env:  []string{versions},
			want: autodetectPlans("nodejs24", "24.22.1"),
		},
		{
			name:  "autodetect_not_enabled",
			files: map[string]string{"package.json": `{"engines": {"node": "24.x"}}`},
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
