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

package gcpbuildpack

import (
	"strings"
	"testing"
	"time"
)

func TestNewSpanValidation(t *testing.T) {
	badName := ""
	_, err := newSpanInfo(badName, time.Now(), time.Now(), nil, StatusOk)
	if err == nil {
		t.Fatalf("newSpanInfo(%q, ...) got err=nil want err!=nil", badName)
	}
	if err.Error() != "span name required" {
		t.Fatalf("newSpanInfo(%q, ...) returned unexpected error %v", badName, err)
	}

	name := "End before start"
	start := time.Now()
	end := start.Add(-1 * time.Second)
	_, err = newSpanInfo(name, start, end, nil, StatusOk)
	if err == nil {
		t.Fatalf("newSpanInfo(%q, ...) got err=nil want err!=nil", name)
	}
	if err.Error() != "start is after end" {
		t.Fatalf("newSpanInfo(%q, ...) returned unexpected error %v", name, err)
	}
}

func TestCreateSpanName(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	testCases := []struct {
		cmd  string
		want string
	}{
		{
			cmd:  "single",
			want: `Exec "single"`,
		},
		{
			cmd:  "one two",
			want: `Exec "one two"`,
		},
		{
			cmd:  "invoke $hello",
			want: `Exec "invoke $hello"`,
		},
		{
			cmd:  "invoke >pipe",
			want: `Exec "invoke >pipe"`,
		},
		{
			cmd:  "invoke --flag && another",
			want: `Exec "invoke --flag && another"`,
		},
		{
			cmd:  `echo "DOUBLE QUOTES"`,
			want: `Exec "echo \"DOUBLE QUOTES\""`,
		},
		{
			cmd:  `echo 'SINGLE QUOTES'`,
			want: `Exec "echo 'SINGLE QUOTES'"`,
		},
		{
			cmd:  "test \r\n\t   characters",
			want: `Exec "test characters"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.cmd, func(t *testing.T) {
			split := strings.Split(tc.cmd, " ")
			got := ctx.createSpanName(split)
			if got != tc.want {
				t.Errorf("CreateSpanName(%q)=%s want=%s", tc.cmd, got, tc.want)
			}
		})
	}
}
