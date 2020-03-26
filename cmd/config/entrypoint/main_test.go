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
)

func TestProcfileWebProcess(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "simple",
			content: "web: foo bar baz",
			want:    "foo bar baz",
		},
		{
			name:    "dollar sign",
			content: "web: foo $bar baz",
			want:    "foo $bar baz",
		},
		{
			name:    "whitespace",
			content: "web:  foo bar baz",
			want:    "foo bar baz",
		},
		{
			name:    "no space",
			content: "web:foo",
			want:    "foo",
		},
		{
			name: "with web in command",
			content: `dev: java --foo=web:something
web: bar baz
`,
			want: "bar baz",
		},
		{
			name: "multiple web use first",
			content: `web: foo
web: bar
`,
			want: "foo",
		},
		{
			name: "multiple web one commented-out",
			content: `# web: foo
web: bar
`,
			want: "bar",
		},
		{
			name:    "trailing newline",
			content: "web: foo bar baz\n",
			want:    "foo bar baz",
		},
		{
			name: "multiple first",
			content: `web:     foo bar
release: baz
dev:     foo
`,
			want: "foo bar",
		},
		{
			name: "multiple middle",
			content: `dev:     foo
web:     bar
release: baz
`,
			want: "bar",
		},
		{
			name: "multiple last",
			content: `dev:     foo
release: bar
web:     baz
`,
			want: "baz",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := procfileWebProcess(tc.content)
			if err != nil {
				t.Fatalf("procfileWebProcess(%s) got error: %v", tc.content, err)
			}
			if got != tc.want {
				t.Errorf("procfileWebProcess(%s) = %q, want %q", tc.content, got, tc.want)
			}
		})
	}
}

func TestProcfileWebProcessError(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "whitespace",
			content: "  web: foo",
		},
		{
			name:    "web in command",
			content: "dev: java --web=foo",
		},
		{
			name:    "no web",
			content: "dev: java",
		},
		{
			name:    "comment",
			content: "# web: java",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := procfileWebProcess(tc.content); err == nil {
				t.Errorf("procfileWebProcess(%s) = %q, want error", tc.content, got)
			}
		})
	}
}
