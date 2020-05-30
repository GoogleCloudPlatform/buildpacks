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
	"os"
	"reflect"
	"strings"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: ".go files",
			files: map[string]string{
				"main.go": "",
			},
			want: 0,
		},
		{
			name:  "no files",
			files: map[string]string{},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetect(t, detectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestGoBuildFlags(t *testing.T) {
	oldEnv := os.Environ()
	t.Cleanup(func() {
		clearAndSetEnv(oldEnv)
	})
	testCases := []struct {
		name     string
		env      []string
		expected []string
	}{
		{
			name:     "no GOOGLE_GOGCFLAGS or GOOGLE_GOLDFLAGS",
			expected: nil,
		},
		{
			name:     "with GOOGLE_GOGCFLAGS",
			env:      []string{"GOOGLE_GOGCFLAGS=gcflags"},
			expected: []string{"-gcflags", "gcflags"},
		},
		{
			name:     "with GOOGLE_GOLDFLAGS",
			env:      []string{"GOOGLE_GOLDFLAGS=ldflags"},
			expected: []string{"-ldflags", "ldflags"},
		},
		{
			name:     "with GOOGLE_GOGCFLAGS and GOOGLE_GOLDFLAGS",
			env:      []string{"GOOGLE_GOGCFLAGS=gcflags1 gcflags2", "GOOGLE_GOLDFLAGS=ldflags1 ldflags2"},
			expected: []string{"-gcflags", "gcflags1 gcflags2", "-ldflags", "ldflags1 ldflags2"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearAndSetEnv(tc.env)
			result := goBuildFlags()
			if !reflect.DeepEqual(tc.expected, result) {
				t.Errorf("goBuildFlags() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func clearAndSetEnv(env []string) {
	os.Clearenv()
	for _, p := range env {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			os.Setenv(kv[0], kv[1])
		}
	}
}
