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
		name string
		env  []string
		want int
	}{
		{
			name: "with app.yaml main",
			env:  []string{"GAE_YAML_MAIN=app.csproj"},
			want: 0,
		},
		{
			name: "without app.yaml main",
			want: 100,
		},
		{
			name: "with empty app.yaml main",
			env:  []string{"GAE_YAML_MAIN="},
			want: 100,
		},
		{
			name: "with GOOGLE_BUILDABLE and app.yaml main",
			env:  []string{"GAE_YAML_MAIN=app.csproj", "GOOGLE_BUILDABLE=other.csproj"},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, map[string]string{}, tc.env, tc.want)
		})
	}
}
