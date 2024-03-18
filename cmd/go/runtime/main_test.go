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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: ".go files and runtime not set",
			files: map[string]string{
				"main.go": "",
			},
			env:  []string{},
			want: 0,
		},
		{
			name:  "no files and runtime not set",
			files: map[string]string{},
			env:   []string{},
			want:  100,
		},
		{
			name: ".go files and runtime set to go",
			files: map[string]string{
				"main.go": "",
			},
			env: []string{
				"GOOGLE_RUNTIME=go",
			},
			want: 0,
		},
		{
			name:  "no .go files and runtime set to go",
			files: map[string]string{},
			env: []string{
				"GOOGLE_RUNTIME=go",
			},
			want: 0,
		},
		{
			name: ".go files and runtime set to non-go",
			files: map[string]string{
				"main.go": "",
			},
			env: []string{
				"GOOGLE_RUNTIME=python",
			},
			want: 100,
		},
		{
			name:  "no .go files and runtime set to non-go",
			files: map[string]string{},
			env: []string{
				"GOOGLE_RUNTIME=python",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestRuntimeVersion(t *testing.T) {
	testCases := []struct {
		name   string
		want   string
		envKey string
	}{
		{name: "GOOGLE_GO_VERSION set",
			envKey: "GOOGLE_GO_VERSION",
			want:   "1.16",
		},

		{name: "GOOGLE_RUNTIME_VERSION set",
			envKey: "GOOGLE_RUNTIME_VERSION",
			want:   "1.16",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				os.Setenv(tc.envKey, tc.want)
			}
			v, err := golang.RuntimeVersion()
			if err != nil {
				t.Fatalf("runtimeVersion() failed: %v", err)
			}
			if v != tc.want {
				t.Errorf("runtimeVersion() = %q, want %q", v, tc.want)
			}

		})
	}
}
