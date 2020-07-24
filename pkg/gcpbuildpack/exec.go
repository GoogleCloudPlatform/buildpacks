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
)

var (
	divider = strings.Repeat("—", 80)
)

// ExecResult bundles exec results.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string
}

type execParams struct {
	cmd         []string
	userFailure bool
	userTiming  bool
	dir         string
	env         []string
	esp         ErrorSummaryProducer
}

type execOption func(o *execParams)

// WithErrorSummaryProducer sets a custom ErrorSummaryProducer.
func WithErrorSummaryProducer(esp ErrorSummaryProducer) execOption {
	return func(o *execParams) {
		o.esp = esp
	}
}

// WithEnv sets environment variables (of the form "KEY=value").
func WithEnv(env ...string) execOption {
	return func(o *execParams) {
		o.env = env
	}
}

// WithWorkDir sets a specific working directory.
func WithWorkDir(dir string) execOption {
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

// Exec runs the given command under the default configuration, handling error if present.
func (ctx *Context) Exec(cmd []string, opts ...execOption) *ExecResult {
	result, err := ctx.ExecWithErr(cmd, opts...)
	if err == nil {
		return result
	}

	ctx.Exit(result.ExitCode, err)
	return nil
}

// ExecWithErr runs the given command (with args) under the default configuration, allowing the caller to handle the error.
func (ctx *Context) ExecWithErr(cmd []string, opts ...execOption) (*ExecResult, *Error) {
	params := execParams{cmd: cmd}
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

	var be *Error
	if result == nil {
		be = Errorf(StatusInternal, err.Error())
	} else {
		if params.esp != nil {
			be = params.esp(result) // TODO: instead of returning an error, just returned the parsed error string. use eo.failure to determine what kind of error to raise.
		} else if params.userFailure {
			be = UserErrorf(result.Combined)
		} else {
			be = Errorf(StatusInternal, result.Combined)
		}
	}
	be.ID = generateErrorID(params.cmd...)
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

	status := StatusInternal
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

	status = StatusOk
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
