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
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

func TestProcfileProcesses(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    []libcnb.Process
	}{
		{
			name:    "simple",
			content: "web: foo bar baz",
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar baz"},
			},
		},
		{
			name:    "dollar sign",
			content: "web: foo $bar baz",
			want: []libcnb.Process{
				{Type: "web", Command: "foo $bar baz"},
			},
		},
		{
			name:    "whitespace start",
			content: "web:  foo bar baz",
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar baz"},
			},
		},
		{
			name:    "whitespace end",
			content: "web:  foo bar baz  ",
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar baz"},
			},
		},
		{
			name:    "carriage return",
			content: "web: foo bar baz\r\n",
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar baz"},
			},
		},
		{
			name:    "no space",
			content: "web:foo",
			want: []libcnb.Process{
				{Type: "web", Command: "foo"},
			},
		},
		{
			name: "multiple with web in Command",
			content: `dev: java --foo=web:something
web: bar baz
`,
			want: []libcnb.Process{
				{Type: "dev", Command: "java --foo=web:something"},
				{Type: "web", Command: "bar baz"},
			},
		},
		{
			name: "multiple web use first",
			content: `web: foo
web: bar
`,
			want: []libcnb.Process{
				{Type: "web", Command: "foo"},
			},
		},
		{
			name: "multiple web one commented-out",
			content: `# web: foo
web: bar
`,
			want: []libcnb.Process{
				{Type: "web", Command: "bar"},
			},
		},
		{
			name:    "trailing newline",
			content: "web: foo bar baz\n",
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar baz"},
			},
		},
		{
			name: "multiple",
			content: `web:     foo bar
release: baz
dev:     foo
`,
			want: []libcnb.Process{
				{Type: "web", Command: "foo bar"},
				{Type: "release", Command: "baz"},
				{Type: "dev", Command: "foo"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContextForTests(libcnb.BuildpackInfo{}, os.TempDir())
			err := addProcfileProcesses(ctx, tc.content)
			if err != nil {
				t.Fatalf("addProcfileProcesses(%s) got error: %v", tc.content, err)
			}
			if got := ctx.Processes(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("addProcfileProcesses(%s) = %#v, want %#v", tc.content, got, tc.want)
			}
		})
	}
}

func TestAddProcfileWebProcessesError(t *testing.T) {
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
			ctx := gcp.NewContextForTests(libcnb.BuildpackInfo{}, os.TempDir())
			if err := addProcfileProcesses(ctx, tc.content); err == nil {
				t.Errorf("procfileWebProcess(%s) = nil, want error", tc.content)
			}
		})
	}
}
