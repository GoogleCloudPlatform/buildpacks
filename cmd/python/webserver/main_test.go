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
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "no py files",
			files: map[string]string{
				"index.js": "",
			},
			want: 0,
		},
		{
			name: "has google entrypoint",
			files: map[string]string{
				"main.py": "",
			},
			env:  []string{"GOOGLE_ENTRYPOINT=gunicorn main:app"},
			want: 100,
		},
		{
			name: "has requirements",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": ""},
			want: 0,
		},
		{
			name: "has gunicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "gunicorn==19.3.0"},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestContainsGunicorn(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gunicorn_present",
			str:  "gunicorn==19.9.0\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_with_comment",
			str:  "gunicorn #my-comment\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_second_line",
			str:  "flask\ngunicorn==19.9.0",
			want: true,
		},
		{
			name: "no_gunicorn_present",
			str:  "gunicorn-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_present",
			str:  "git+git://github.com/gunicorn@master#egg=gunicorn\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_egg_not_present",
			str:  "git+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nflask\n",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsGunicorn(tc.str)
			if got != tc.want {
				t.Errorf("containsGunicorn() got %t, want %t", got, tc.want)
			}
		})
	}
}
