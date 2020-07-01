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
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestExecEmitsSpan(t *testing.T) {
	cmd := []string{"echo", "Hello"}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	ctx.ExecWithErr(cmd)

	if len(ctx.stats.spans) != 1 {
		t.Fatalf("Unexpected number of spans, got %d want 1", len(ctx.stats.spans))
	}
	span := ctx.stats.spans[0]
	wantSpanName := `Exec "echo Hello"`
	if span.name != wantSpanName {
		t.Errorf("Unexpected span name got %q want %q", span.name, wantSpanName)
	}
	if span.status != StatusOk {
		t.Errorf("Unexpected span status got %d want %d", span.status, StatusOk)
	}
}

func TestExecWithErrInvokesCommand(t *testing.T) {
	cmd := []string{"echo", "Hello"}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	result, err := ctx.ExecWithErr(cmd)
	if err != nil {
		t.Errorf("ExecWithErr(%v) got unexpected error: %v", cmd, err)
	}
	want := "Hello"
	if result.Stdout != want {
		t.Errorf("ExecWithErr(%v) got stdout=%q, want stdout=%q", cmd, result.Stdout, want)
	}
}

func TestExecInvokesCommand(t *testing.T) {
	cmd := []string{"echo", "Hello"}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	result := ctx.Exec(cmd)
	want := "Hello"
	if result.Stdout != want {
		t.Errorf("Exec(%v) got stdout=%q, want stdout=%q", cmd, result.Stdout, want)
	}
}

func TestExecResult(t *testing.T) {
	cmd := []string{"/bin/bash", "-f", "-c", "printf 'stdout'; printf 'stderr' >&2"}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	got := ctx.Exec(cmd)

	if got.ExitCode != 0 {
		t.Error("Exit code got 0, want != 0")
	}
	if got.Stdout != "stdout" {
		t.Errorf("stdout got %q, want `out`", got.Stdout)
	}
	if got.Stderr != "stderr" {
		t.Errorf("stderr got %q, want `err`", got.Stderr)
	}
	// Combined may be some arbitrary interleaving of stdout/stderr.
	if !hasInterleavedString(t, got.Combined, "out") {
		t.Errorf("Combined %q does not contain interleaved `out`", got.Combined)
	}
	if !hasInterleavedString(t, got.Combined, "err") {
		t.Errorf("Combined %q does not contain interleaved `err`", got.Combined)
	}
}

func hasInterleavedString(t *testing.T, s, sub string) bool {
	t.Helper()

	// Build a regex that allows any letters to be interleaved.
	re := ".*" + strings.Join(strings.Split(sub, ""), ".*") + ".*"

	match, err := regexp.MatchString(re, s)
	if err != nil {
		t.Fatalf("Matching %q: %v", re, err)
	}
	return match
}

func TestHasInterleavedString(t *testing.T) {
	testCases := []struct {
		name  string
		s     string
		sub   string
		match bool
	}{
		{
			name:  "exact",
			s:     "abc",
			sub:   "abc",
			match: true,
		},
		{
			name:  "substr",
			s:     "---abc",
			sub:   "abc",
			match: true,
		},
		{
			name:  "interleaved",
			s:     "-ab--c",
			sub:   "abc",
			match: true,
		},
		{
			name:  "chars present but out of order",
			s:     "-ac--b",
			sub:   "abc",
			match: false,
		},
		{
			name:  "too short",
			s:     "a",
			sub:   "abc",
			match: false,
		},
		{
			name:  "empty",
			s:     "",
			sub:   "abc",
			match: false,
		},
		{
			name:  "empty sub",
			s:     "abc",
			sub:   "",
			match: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasInterleavedString(t, tc.s, tc.sub)

			if got != tc.match {
				t.Errorf("hasInterleavedString(%q, %q)=%t, want=%t", tc.s, tc.sub, got, tc.match)
			}
		})
	}
}

func TestExecUserUpdatesDuration(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	dur := ctx.stats.user
	if dur != 0 {
		t.Fatalf("User duration is not zero to start")
	}

	ctx.ExecUser([]string{"sleep", ".1"})
	if ctx.stats.user <= dur {
		t.Errorf("ExecUser(): user duration did not increase")
	}

	dur = ctx.stats.user
	ctx.ExecUserWithParams(ExecParams{Cmd: []string{"sleep", ".1"}}, nil)
	if ctx.stats.user <= dur {
		t.Errorf("ExecUserWithParams(): user duration did not increase")
	}
	if ctx.stats.user < 200*time.Millisecond {
		t.Errorf("ExecUserWithParams(): user duration did not increase enough, got %s, want >= %s", ctx.stats.user, 200*time.Millisecond)
	}
}

func TestExecDoesNotUpdateDuration(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	dur := ctx.stats.user
	if dur != 0 {
		t.Fatalf("User duration is not zero to start")
	}

	ctx.Exec([]string{"sleep", ".1"})
	if ctx.stats.user != 0 {
		t.Fatalf("Exec(): user duration changed unexpectedly")
	}

	ctx.ExecWithParams(ExecParams{Cmd: []string{"sleep", ".1"}})
	if ctx.stats.user != 0 {
		t.Fatalf("ExecWithParams(): user duration changed unexpectedly")
	}
}
