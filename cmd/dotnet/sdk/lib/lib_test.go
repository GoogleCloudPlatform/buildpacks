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
			name: "csproj",
			files: map[string]string{
				"app.csproj": "",
			},
			want: 0,
		},
		{
			name: "csproj with runtime set to dotnet",
			files: map[string]string{
				"app.csproj": "",
			},
			env:  []string{"GOOGLE_RUNTIME=dotnet"},
			want: 0,
		},
		{
			name: "csproj with runtime set to python",
			files: map[string]string{
				"app.csproj": "",
			},
			env:  []string{"GOOGLE_RUNTIME=python"},
			want: 100,
		},
		{
			name: "fsproj",
			files: map[string]string{
				"app.fsproj": "",
			},
			want: 0,
		},
		{
			name: "vbproj",
			files: map[string]string{
				"app.vbproj": "",
			},
			want: 0,
		},
		{
			name: "unsupported pyproj",
			files: map[string]string{
				".pyproj": "",
			},
			want: 100,
		},
		{
			name: "unsupported partly matching",
			files: map[string]string{
				"app.mycsproj": "",
			},
			want: 100,
		},
		{
			name: "without project files",
			files: map[string]string{
				"Program.cs": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}
