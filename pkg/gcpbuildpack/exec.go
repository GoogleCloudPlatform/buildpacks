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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
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

// ExecParams bundles exec parameters.
type ExecParams struct {
	// Cmd is the required command and optional arguments to be run.
	Cmd []string
	// Dir identifies the directory in which the command should be run. Default to the current working directory.
	Dir string
	// Env specifies additional environment variables for the command invocation. Must be in key=value format.
	Env []string

	// logOnDebug indicates that the logs will be emitted only if GOOGLE_DEBUG is set (otherwise logs are always emitted).
	logOnDebug bool
}

// Exec runs the given command under the default configuration, handling error if present.
// Exec failures attribute the failure to the platform, not the user, when recording the error (see builderoutput.go).
func (ctx *Context) Exec(cmd []string) *ExecResult {
	return ctx.ExecWithParams(ExecParams{Cmd: cmd})
}

// ExecWithParams runs the given command under the specified configuration, handling the error if present.
// ExecWithParams failures attribute the failure to the platform, not the user, when recording the error (see builderoutput.go).
func (ctx *Context) ExecWithParams(params ExecParams) *ExecResult {
	result, err := ctx.ExecWithErrWithParams(params)
	if err != nil {
		var be *Error
		exitCode := 1
		if result == nil {
			be = Errorf(StatusInternal, err.Error())
		} else {
			be = Errorf(StatusInternal, result.Combined)
			exitCode = result.ExitCode
		}
		be.ID = generateErrorID(params.Cmd...)
		ctx.Exit(exitCode, be)
	}
	return result
}

// ExecWithErr runs the given command (with args) under the default configuration, allowing the caller to handle the error.
func (ctx *Context) ExecWithErr(cmd []string) (*ExecResult, error) {
	return ctx.ExecWithErrWithParams(ExecParams{Cmd: cmd})
}

// ExecWithErrWithParams runs the given command (with args) under the specified configuration, allowing the caller to handle the error.
func (ctx *Context) ExecWithErrWithParams(params ExecParams) (*ExecResult, error) {
	params.logOnDebug = true
	return ctx.configuredExec(params)
}

// ExecUser runs the given command under the default configuration, saving the tail of stderr.
// ExecUser failures attribute the failure to the user, not the platform, when recording the error (see builderoutput.go).
func (ctx *Context) ExecUser(cmd []string) *ExecResult {
	return ctx.ExecUserWithParams(ExecParams{Cmd: cmd}, UserErrorKeepStderrTail)
}

// ExecUserWithParams runs the given command under the specified configuration, saving an error summary from producer on error.
// ExecUserWithParams failures attribute the failure to the user, not the platform, when recording the error (see builderoutput.go).
func (ctx *Context) ExecUserWithParams(params ExecParams, esp ErrorSummaryProducer) *ExecResult {
	start := time.Now()
	result, err := ctx.configuredExec(params)
	if err != nil {
		var be *Error
		if result == nil {
			be = Errorf(StatusInternal, err.Error())
		} else {
			be = esp(result)
		}
		be.ID = generateErrorID(params.Cmd...)
		ctx.Exit(1, be)
	}
	ctx.stats.user += time.Since(start)
	return result
}

func (ctx *Context) configuredExec(params ExecParams) (*ExecResult, error) {
	if len(params.Cmd) < 1 {
		return nil, fmt.Errorf("no command provided")
	}
	if params.Cmd[0] == "" {
		return nil, fmt.Errorf("empty command provided")
	}

	log := true
	debug, err := env.IsDebugMode()
	if err != nil {
		return nil, err
	}
	if params.logOnDebug && !debug {
		log = false
	}

	optionalLogf := func(format string, args ...interface{}) {
		if !log {
			return
		}
		ctx.Logf(format, args...)
	}

	readableCmd := strings.Join(params.Cmd, " ")
	optionalLogf(divider)
	optionalLogf("Running %q", readableCmd)

	status := StatusInternal
	defer func(start time.Time) {
		truncated := readableCmd
		if len(truncated) > 60 {
			truncated = truncated[:60] + "..."
		}
		optionalLogf("Done %q (%v)", truncated, time.Since(start))
		ctx.Span(ctx.createSpanName(params.Cmd), start, status)
	}(time.Now())

	exitCode := 0
	ecmd := exec.Command(params.Cmd[0], params.Cmd[1:]...)

	if params.Dir != "" {
		ecmd.Dir = params.Dir
	}

	if len(params.Env) > 0 {
		ecmd.Env = append(os.Environ(), params.Env...)
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
