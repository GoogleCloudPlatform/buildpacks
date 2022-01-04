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

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
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

func TestJSONVersionParse(t *testing.T) {
	testCases := []struct {
		name string
		want string
		json string
	}{
		{
			name: "all_stable",
			want: "1.16",
			json: `
[
 {
  "version": "go1.16",
  "stable": true
 },
 {
  "version": "go1.15.3",
  "stable": true
 },
 {
  "version": "go1.12.12",
  "stable": true
 }
]`,
		},
		{
			name: "recent_unstable",
			want: "1.15.3",
			json: `
[
 {
  "version": "go1.15.4",
  "stable": false
 },
 {
  "version": "go1.15.3",
  "stable": true
 },
 {
  "version": "go1.12.12",
  "stable": true
 }
]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if v, err := parseVersionJSON(tc.json); err != nil {
				t.Fatalf("parseVersionJSON() failed: %v", tc.name, err)
			} else if v != tc.want {
				t.Errorf("parseVersionJSON() = %q, want %q", tc.name, v, tc.want)
			}
		})
	}
}
