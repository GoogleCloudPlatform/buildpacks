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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

func TestExecEmitsSpan(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	ctx.Exec(strings.Fields("echo Hello"))

	if len(ctx.stats.spans) != 1 {
		t.Fatalf("Unexpected number of spans, got %d want 1", len(ctx.stats.spans))
	}
	span := ctx.stats.spans[0]
	wantSpanName := `Exec "echo Hello"`
	if span.name != wantSpanName {
		t.Errorf("Unexpected span name got %q want %q", span.name, wantSpanName)
	}
	if span.status != buildererror.StatusOk {
		t.Errorf("Unexpected span status got %d want %d", span.status, buildererror.StatusOk)
	}
}

func TestExecInvokesCommand(t *testing.T) {
	cmd := strings.Fields("echo Hello")
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	result, err := ctx.Exec(cmd)
	if err != nil {
		t.Errorf("Exec(%v) got unexpected error: %v", cmd, err)
	}
	want := "Hello"
	if result.Stdout != want {
		t.Errorf("Exec(%v) got stdout=%q, want stdout=%q", cmd, result.Stdout, want)
	}
}

func TestExecResult(t *testing.T) {
	cmd := []string{"/bin/bash", "-f", "-c", "printf 'stdout'; printf 'stderr' >&2"}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	got, err := ctx.Exec(cmd)

	if err != nil {
		t.Fatalf("Exec(%v) got unexpected error: %v", cmd, err)
	}
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

func TestExecAsUserUpdatesDuration(t *testing.T) {
	testCases := []struct {
		name string
		opt  func(*execParams)
	}{
		{name: "WithUserTimingAttribution", opt: WithUserTimingAttribution},
		{name: "WithUserAttribution", opt: WithUserAttribution},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cleanUp := simpleContext(t)
			defer cleanUp()

			dur := ctx.stats.user
			if dur != 0 {
				t.Fatalf("user duration is not zero to start")
			}

			ctx.Exec(strings.Fields("sleep .1"), tc.opt)
			if ctx.stats.user <= dur {
				t.Errorf("user duration did not increase")
			}
		})
	}
}

func TestExecAsDefaultDoesNotUpdateDuration(t *testing.T) {
	testCases := []struct {
		name string
		opt  func(*execParams)
	}{
		{name: "default"},
		{name: "WithUserFailureAttribution", opt: WithUserFailureAttribution}, // WithUserFailureAttribution should not impact timing attribution.
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cleanUp := simpleContext(t)
			defer cleanUp()
			opts := []ExecOption{}
			if tc.opt != nil {
				opts = append(opts, tc.opt)
			}

			dur := ctx.stats.user
			if dur != 0 {
				t.Fatalf("User duration is not zero to start")
			}

			ctx.Exec(strings.Fields("sleep .1"), opts...)
			if ctx.stats.user != 0 {
				t.Fatalf("Exec(): user duration changed unexpectedly")
			}
		})
	}
}

func (ctx *Context) execWithErrCastToBuildError(cmd []string, opts ...ExecOption) (*ExecResult, *buildererror.Error) {
	result, err := ctx.Exec(cmd, opts...)
	if err == nil {
		return result, nil
	}
	return result, err.(*buildererror.Error)
}

func TestExecAsUserDoesNotReturnStatusInternal(t *testing.T) {
	testCases := []struct {
		name string
		opt  func(*execParams)
	}{
		{name: "WithUserFailureAttribution", opt: WithUserFailureAttribution},
		{name: "WithUserAttribution", opt: WithUserAttribution},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cleanUp := simpleContext(t)
			defer cleanUp()

			result, err := ctx.execWithErrCastToBuildError([]string{"/bin/bash", "-c", "exit 99"}, tc.opt)

			if err.Status == buildererror.StatusInternal {
				t.Error("unexpected error status StatusInternal")
			}
			if got, want := result.ExitCode, 99; got != want {
				t.Errorf("incorrect exit code got %d want %d", got, want)
			}
		})
	}
}

func TestExecAsDefaultReturnsStatusInternal(t *testing.T) {
	testCases := []struct {
		name string
		opt  func(*execParams)
	}{
		{name: "default"},
		{name: "WithUserTimingAttribution", opt: WithUserTimingAttribution}, // WithUserTimingAttribution should not impact failure attribution.
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cleanUp := simpleContext(t)
			defer cleanUp()
			opts := []ExecOption{}
			if tc.opt != nil {
				opts = append(opts, tc.opt)
			}

			result, err := ctx.execWithErrCastToBuildError([]string{"/bin/bash", "-c", "exit 99"}, opts...)

			if got, want := err.Status, buildererror.StatusInternal; got != want {
				t.Errorf("incorrect error status got %v want %v", got, want)
			}
			if got, want := result.ExitCode, 99; got != want {
				t.Errorf("incorrect exit code got %d want %d", got, want)
			}
		})
	}
}

func TestExecWithEnv(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	cmd := []string{"/bin/bash", "-c", "echo $FOO"}

	result, err := ctx.Exec(cmd, WithEnv("A=B", "FOO=bar"))

	if err != nil {
		t.Fatalf("Exec(%v) got unexpected error: %v", cmd, err)
	}
	if got, want := strings.TrimSpace(result.Stdout), "bar"; got != want {
		t.Errorf("incorrect output got=%q want=%q", got, want)
	}
}

func TestExecWithEnvMultiple(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	cmd := []string{"/bin/bash", "-c", "echo $A $FOO"}

	result, err := ctx.Exec(cmd, WithEnv("A=B", "FOO=bar"), WithEnv("FOO=baz"))

	if err != nil {
		t.Fatalf("Exec(%v) got unexpected error: %v", cmd, err)
	}
	if got, want := strings.TrimSpace(result.Stdout), "B baz"; got != want {
		t.Errorf("incorrect output got=%q want=%q", got, want)
	}
}

func TestExecWithWorkDir(t *testing.T) {
	tdir, err := ioutil.TempDir("", "exec2-")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	cmd := []string{"/bin/bash", "-c", "echo $PWD"}

	result, err := ctx.Exec(cmd, WithWorkDir(tdir))

	if err != nil {
		t.Fatalf("Exec(%v) got unexpected error: %v", cmd, err)
	}
	if got, want := strings.TrimSpace(result.Stdout), tdir; got != want {
		t.Errorf("incorrect output got=%q want=%q", got, want)
	}
}

func TestExecWithMessageProducer(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()
	wantProducer := func(result *ExecResult) string { return "HELLO" }

	_, gotErr := ctx.execWithErrCastToBuildError([]string{"/bin/bash", "-c", "exit 99"}, WithMessageProducer(wantProducer))

	if got, want := gotErr.Message, "HELLO"; got != want {
		t.Errorf("incorrect message got=%q want=%q", got, want)
	}
}

func TestMessageProducerHelpers(t *testing.T) {
	testCases := []struct {
		name     string
		opt      ExecOption
		stdout   string
		stderr   string
		combined string
		want     string
	}{
		{
			name:   "WithStdoutTail",
			opt:    WithStdoutTail,
			stdout: "123456789stdout",
			want:   "...stdout",
		},
		{
			name:   "WithStderrTail",
			opt:    WithStderrTail,
			stderr: "123456789stderr",
			want:   "...stderr",
		},
		{
			name:     "WithCombinedTail",
			opt:      WithCombinedTail,
			combined: "123456789combined",
			want:     "...mbined",
		},
		{
			name:   "WithStdoutHead",
			opt:    WithStdoutHead,
			stdout: "stdout123456789",
			want:   "stdout...",
		},
		{
			name:   "WithStderrHead",
			opt:    WithStderrHead,
			stderr: "stderr123456789",
			want:   "stderr...",
		},
		{
			name:     "WithCombinedHead",
			opt:      WithCombinedHead,
			combined: "combined123456789",
			want:     "combin...",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldMax := maxMessageBytes
			maxMessageBytes = 9
			defer func() {
				maxMessageBytes = oldMax
			}()

			ep := execParams{}
			tc.opt(&ep)
			got := ep.messageProducer(&ExecResult{Stdout: tc.stdout, Stderr: tc.stderr, Combined: tc.combined})

			if got != tc.want {
				t.Errorf("message got %q want %q", got, tc.want)
			}
		})
	}
}

func TestExec(t *testing.T) {
	testCases := []struct {
		name            string
		cmd             []string
		opts            []ExecOption
		wantResult      *ExecResult
		wantErr         bool
		wantErrMessage  string
		wantUserTiming  bool
		wantMinUserDur  time.Duration
		wantUserFailure bool
	}{
		{
			name:           "nil cmd",
			wantErr:        true,
			wantErrMessage: "no command provided",
		},
		{
			name:           "empty cmd slice",
			cmd:            []string{},
			wantErr:        true,
			wantErrMessage: "no command provided",
		},
		{
			name:           "empty cmd",
			cmd:            []string{""},
			wantErr:        true,
			wantErrMessage: "empty command provided",
		},
		{
			name:           "successful cmd with user attribution",
			cmd:            []string{"sleep", ".5"},
			opts:           []ExecOption{WithUserAttribution},
			wantResult:     &ExecResult{},
			wantUserTiming: true,
			wantMinUserDur: 500 * time.Millisecond,
		},
		{
			name:           "successful cmd with user timing attribution",
			cmd:            []string{"sleep", ".5"},
			opts:           []ExecOption{WithUserTimingAttribution},
			wantResult:     &ExecResult{},
			wantUserTiming: true,
			wantMinUserDur: 500 * time.Millisecond,
		},
		{
			name:           "successful cmd with user failure attribution",
			cmd:            []string{"sleep", ".5"},
			opts:           []ExecOption{WithUserFailureAttribution},
			wantResult:     &ExecResult{},
			wantUserTiming: false,
		},
		{
			name:            "failing cmd with user attribution",
			cmd:             []string{"bash", "-c", "sleep .5; exit 99"},
			opts:            []ExecOption{WithUserAttribution},
			wantResult:      &ExecResult{ExitCode: 99},
			wantErr:         true,
			wantUserTiming:  true,
			wantMinUserDur:  500 * time.Millisecond,
			wantUserFailure: true,
		},
		{
			name:           "failing cmd with user timing attribution",
			cmd:            []string{"bash", "-c", "sleep .5; exit 99"},
			opts:           []ExecOption{WithUserTimingAttribution},
			wantResult:     &ExecResult{ExitCode: 99},
			wantErr:        true,
			wantUserTiming: true,
			wantMinUserDur: 500 * time.Millisecond,
		},
		{
			name:            "failing cmd with user failure attribution",
			cmd:             []string{"bash", "-c", "sleep .5; exit 99"},
			opts:            []ExecOption{WithUserFailureAttribution},
			wantResult:      &ExecResult{ExitCode: 99},
			wantErr:         true,
			wantUserTiming:  false,
			wantUserFailure: true,
		},
		{
			name:           "enoent cmd with user failure attribution",
			cmd:            []string{"cat", "/tmp/does-not-exist-123456"},
			opts:           []ExecOption{WithUserAttribution},
			wantErrMessage: "...ory", // Last few characters of message below due to maxMessageBytes setting in main test below.
			wantResult: &ExecResult{
				ExitCode: 1,
				Stderr:   "cat: /tmp/does-not-exist-123456: No such file or directory",
				Combined: "cat: /tmp/does-not-exist-123456: No such file or directory",
			},
			wantErr:         true,
			wantUserTiming:  true,
			wantUserFailure: true,
		},
		{
			name:       "WithEnv",
			cmd:        []string{"bash", "-c", "echo $FOO"},
			opts:       []ExecOption{WithEnv("FOO=bar")},
			wantResult: &ExecResult{Stdout: "bar", Combined: "bar"},
		},
		{
			name:       "WithWorkDir",
			cmd:        []string{"bash", "-c", "echo $PWD"},
			opts:       []ExecOption{WithWorkDir(os.TempDir())},
			wantResult: &ExecResult{Stdout: os.TempDir(), Combined: os.TempDir()},
		},
		{
			name:           "WithMessageProducer",
			cmd:            []string{"bash", "-c", "exit 99"},
			opts:           []ExecOption{WithMessageProducer(func(result *ExecResult) string { return "foo" })},
			wantErr:        true,
			wantErrMessage: "foo",
			wantResult:     &ExecResult{ExitCode: 99},
		},
		{
			name:           "WithStdoutTail",
			cmd:            []string{"bash", "-c", "echo ------foo; exit 99"},
			opts:           []ExecOption{WithStdoutTail},
			wantErr:        true,
			wantErrMessage: "...foo",
			wantResult:     &ExecResult{ExitCode: 99, Stdout: "------foo", Combined: "------foo"},
		},
		{
			name:           "WithStdoutHead",
			cmd:            []string{"bash", "-c", "echo foo------; exit 99"},
			opts:           []ExecOption{WithStdoutHead},
			wantErr:        true,
			wantErrMessage: "foo...",
			wantResult:     &ExecResult{ExitCode: 99, Stdout: "foo------", Combined: "foo------"},
		},
		{
			name:           "WithStderrTail",
			cmd:            []string{"bash", "-c", "echo ------foo >&2; exit 99"},
			opts:           []ExecOption{WithStderrTail},
			wantErr:        true,
			wantErrMessage: "...foo",
			wantResult:     &ExecResult{ExitCode: 99, Stderr: "------foo", Combined: "------foo"},
		},
		{
			name:           "WithStderrHead",
			cmd:            []string{"bash", "-c", "echo foo------ >&2; exit 99"},
			opts:           []ExecOption{WithStderrHead},
			wantErr:        true,
			wantErrMessage: "foo...",
			wantResult:     &ExecResult{ExitCode: 99, Stderr: "foo------", Combined: "foo------"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldMaxMessageBytes := maxMessageBytes
			maxMessageBytes = 6
			defer func() {
				maxMessageBytes = oldMaxMessageBytes
			}()
			ctx := NewContext()

			result, err := ctx.execWithErrCastToBuildError(tc.cmd, tc.opts...)

			if got, want := err != nil, tc.wantErr; got != want {
				t.Errorf("got error %t want error %t", got, want)
			}
			if !reflect.DeepEqual(result, tc.wantResult) {
				t.Errorf("incorrect result got %#v want %#v", result, tc.wantResult)
			}

			if tc.wantUserTiming && ctx.stats.user < tc.wantMinUserDur {
				t.Errorf("got user timing %v want timing >= %v", ctx.stats.user, tc.wantMinUserDur)
			}
			if !tc.wantUserTiming && ctx.stats.user > 0 {
				t.Error("got user timing > 0, want user timing 0")
			}

			if tc.wantErr {
				if tc.wantUserFailure {
					if err.Status == buildererror.StatusInternal {
						t.Error("got error status internal (i.e., system attribution), want something else")
					}
				} else {
					if err.Status != buildererror.StatusInternal {
						t.Errorf("got error status %s, want status internal", err.Status)
					}
				}
				if err.Message != tc.wantErrMessage {
					t.Errorf("incorrect error message got %q want %q", err.Message, tc.wantErrMessage)
				}
				if err.ID == "" {
					t.Errorf("missing error ID")
				}
			}
		})
	}
}

func TestExecWithCRLF(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only applicable for Linux")
	}

	f := filepath.Join(t.TempDir(), "script")
	// must be executable
	if err := ioutil.WriteFile(f, []byte("#!/bin/sh\r\necho NO\r\n"), 0555); err != nil {
		t.Fatal(err)
	}

	ctx := NewContext()
	_, gotErr := ctx.execWithErrCastToBuildError([]string{f})
	if gotErr == nil {
		t.Errorf("expected error: %v", gotErr)
	} else if !strings.Contains(gotErr.Message, "Unix-style LF") {
		t.Errorf("should have mentioned Unix line-endings: %v", gotErr)
	}
}

type fakeExiter struct {
	called bool
	code   int
	err    *buildererror.Error
}

func (e *fakeExiter) Exit(exitCode int, be *buildererror.Error) {
	e.called = true
	e.code = exitCode
	e.err = be
}
