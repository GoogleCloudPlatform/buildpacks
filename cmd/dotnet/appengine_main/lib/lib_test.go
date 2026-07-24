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
		name string
		env  []string
		want int
	}{
		{
			name: "would normally opt in, but without TARGET_PLATFORM should opt out",
			env:  []string{"GAE_YAML_MAIN=app.csproj"},
			want: 100,
		},
		{
			name: "with app.yaml main",
			env:  []string{"GAE_YAML_MAIN=app.csproj", "X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 0,
		},
		{
			name: "without app.yaml main",
			env:  []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 100,
		},
		{
			name: "with empty app.yaml main",
			env:  []string{"GAE_YAML_MAIN=", "X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 100,
		},
		{
			name: "with GOOGLE_BUILDABLE and app.yaml main",
			env:  []string{"GAE_YAML_MAIN=app.csproj", "GOOGLE_BUILDABLE=other.csproj", "X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, map[string]string{}, tc.env, tc.want)
		})
	}
}
