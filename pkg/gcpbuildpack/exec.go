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
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"golang.org/x/sys/unix"
)

var (
	divider = strings.Repeat("-", 80)
)

// ExecResult bundles exec results.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string
}

type execParams struct {
	cmd []string
	dir string
	env []string

	userFailure     bool
	userTiming      bool
	messageProducer MessageProducer
}

// ExecOption configures Exec functions.
type ExecOption func(o *execParams)

// WithEnv sets environment variables (of the form "KEY=value").
func WithEnv(env ...string) ExecOption {
	return func(o *execParams) {
		o.env = append(o.env, env...)
	}
}

// WithWorkDir sets a specific working directory.
func WithWorkDir(dir string) ExecOption {
	return func(o *execParams) {
		o.dir = dir
	}
}

// WithUserAttribution indicates that failure and timing both are attributed to the user.
var WithUserAttribution = func(o *execParams) {
	o.userFailure = true
	o.userTiming = true
}

// WithUserTimingAttribution indicates that only timing is attributed to the user.
var WithUserTimingAttribution = func(o *execParams) {
	o.userTiming = true
}

// WithUserFailureAttribution indicates that only failure is attributed to the user.
var WithUserFailureAttribution = func(o *execParams) {
	o.userFailure = true
}

// WithMessageProducer sets a custom MessageProducer to produce the error message.
func WithMessageProducer(mp MessageProducer) ExecOption {
	return func(o *execParams) {
		o.messageProducer = mp
	}
}

// WithCombinedTail keeps the tail of the combined stdout/stderr for the error message.
var WithCombinedTail = WithMessageProducer(KeepCombinedTail)

// WithCombinedHead keeps the head of the combined stdout/stderr for the error message.
var WithCombinedHead = WithMessageProducer(KeepCombinedHead)

// WithStderrTail keeps the tail of stderr for the error message.
var WithStderrTail = WithMessageProducer(KeepStderrTail)

// WithStderrHead keeps the head of stderr for the error message.
var WithStderrHead = WithMessageProducer(KeepStderrHead)

// WithStdoutTail keeps the tail of stdout for the error message.
var WithStdoutTail = WithMessageProducer(KeepStdoutTail)

// WithStdoutHead keeps the head of stdout for the error message.
var WithStdoutHead = WithMessageProducer(KeepStdoutHead)

// Exec runs the given command under the default configuration, handling error if present.
func (ctx *Context) Exec(cmd []string, opts ...ExecOption) *ExecResult {
	result, err := ctx.ExecWithErr(cmd, opts...)
	if err == nil {
		return result
	}

	exitCode := 1
	if result != nil {
		exitCode = result.ExitCode
	}
	ctx.Exit(exitCode, err)
	return nil
}

// ExecWithErr runs the given command (with args) under the default configuration, allowing the caller to handle the error.
func (ctx *Context) ExecWithErr(cmd []string, opts ...ExecOption) (*ExecResult, *buildererror.Error) {
	params := execParams{cmd: cmd, messageProducer: KeepCombinedTail}
	for _, o := range opts {
		o(&params)
	}

	start := time.Now()

	result, err := ctx.configuredExec(params)

	if params.userTiming {
		ctx.stats.user += time.Since(start)
	}

	if err == nil {
		return result, nil
	}

	message := err.Error()
	if result != nil {
		message = params.messageProducer(result)
	}

	var be *buildererror.Error
	if params.userFailure {
		be = UserErrorf(message)
	} else {
		be = buildererror.Errorf(buildererror.StatusInternal, message)
	}

	be.ID = buildererror.GenerateErrorID(params.cmd...)
	return result, be
}

func (ctx *Context) configuredExec(params execParams) (*ExecResult, error) {
	if len(params.cmd) < 1 {
		return nil, fmt.Errorf("no command provided")
	}
	if params.cmd[0] == "" {
		return nil, fmt.Errorf("empty command provided")
	}

	log := true
	if !params.userFailure && !ctx.debug {
		// For "system" commands, we will only log if the debug flag is present.
		log = false
	}

	optionalLogf := func(format string, args ...interface{}) {
		if !log {
			return
		}
		ctx.Logf(format, args...)
	}

	readableCmd := strings.Join(params.cmd, " ")
	if len(params.env) > 0 {
		env := strings.Join(params.env, " ")
		readableCmd = fmt.Sprintf("%s (%s)", readableCmd, env)
	}
	optionalLogf(divider)
	optionalLogf("Running %q", readableCmd)

	status := buildererror.StatusInternal
	defer func(start time.Time) {
		truncated := readableCmd
		if len(truncated) > 60 {
			truncated = truncated[:60] + "..."
		}
		optionalLogf("Done %q (%v)", truncated, time.Since(start))
		ctx.Span(ctx.createSpanName(params.cmd), start, status)
	}(time.Now())

	exitCode := 0
	ecmd := exec.Command(params.cmd[0], params.cmd[1:]...)

	if params.dir != "" {
		ecmd.Dir = params.dir
	}

	if len(params.env) > 0 {
		ecmd.Env = append(os.Environ(), params.env...)
	}

	var outb, errb bytes.Buffer
	combinedb := lockingBuffer{log: log}
	ecmd.Stdout = io.MultiWriter(&outb, &combinedb)
	ecmd.Stderr = io.MultiWriter(&errb, &combinedb)

	if err := ecmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// The command returned a non-zero result.
			exitCode = ee.ExitCode()
		} else if pe, ok := err.(*os.PathError); ok && pe.Err == unix.ENOENT {
			// ENOENT normally occurs if the command cannot
			// be found, but also occurs with scripts using
			// CR-LF line endings.  Unix uses LF as its line
			// ending, so a script with a shebang using CR-LF
			// will result in the kernel attempting to
			// resolve an executable name with the trailing
			// CR. This search will almost certainly fail and
			// otherwise results in an confusing ENOENT.
			return nil, fmt.Errorf("executing command %q: %v: if %q is a script, ensure that it has Unix-style LF line endings", readableCmd, err, params.cmd[0])
		} else {
			return nil, fmt.Errorf("executing command %q: %v", readableCmd, err)
		}
	}

	result := &ExecResult{
		ExitCode: exitCode,
		Stdout:   strings.TrimSpace(string(outb.Bytes())),
		Stderr:   strings.TrimSpace(string(errb.Bytes())),
		Combined: strings.TrimSpace(string(combinedb.Bytes())),
	}

	if exitCode != 0 {
		return result, fmt.Errorf("executing command %q: exit code %d", readableCmd, exitCode)
	}

	status = buildererror.StatusOk
	return result, nil
}

type lockingBuffer struct {
	buf bytes.Buffer
	sync.Mutex

	// log tells the buffer to also log the output to stderr.
	log bool
}

func (lb *lockingBuffer) Write(p []byte) (int, error) {
	lb.Lock()
	defer lb.Unlock()
	if lb.log {
		os.Stderr.Write(p)
	}
	return lb.buf.Write(p)
}

func (lb *lockingBuffer) Bytes() []byte {
	return lb.buf.Bytes()
}
