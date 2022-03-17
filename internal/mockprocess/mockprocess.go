// Copyright 2022 Google LLC
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

// Package mockprocess provides testing utilities to mock out exec.Cmd
// shell commands with a mock process.
package mockprocess

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess/mockprocessutil"
)

// Mock associates a mock process with a command.
type Mock struct {
	commandRegex  string
	processConfig *mockprocessutil.MockProcessConfig
}

// New mocks the behavior of a shell command executed by a
// ctx.Exec call. `commandRegex` is the command to mock; the regex must match
// the full command that would have been executed, though it
// does not have to be the beginning of the command. `stdout` is what will
// be printed to stdout, `stderr` is what wiil be printed totderr. `exitCode`
// will be the exit code of the command.
//
// All commands executed through ctx.Exec have stdout and stderr redirected
// to the return parameters of ctx.Exec. However, the combined output ends
// up being logged to stderr of the parent process. The stderr of executing
// detectFn or buildFn can be searched for the stdout or stderr of any ctx.Exec
// mocks.
func New(commandRegex string, opts ...Option) *Mock {
	mp := &mockprocessutil.MockProcessConfig{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
	}
	for _, o := range opts {
		o(mp)
	}
	return &Mock{
		commandRegex:  commandRegex,
		processConfig: mp,
	}
}

// Option are options that configure the behavior of the mock command
// that replaces ctx.Exec calls.
type Option func(*mockprocessutil.MockProcessConfig)

// WithStdout configures what a mocked command prints to stdout.
func WithStdout(msg string) Option {
	return func(mp *mockprocessutil.MockProcessConfig) {
		mp.Stdout = msg
	}
}

// WithStderr configures what a mocked command prints to stderr.
func WithStderr(msg string) Option {
	return func(mp *mockprocessutil.MockProcessConfig) {
		mp.Stderr = msg
	}
}

// WithExitCode configures what a mocked command uses as the exit code.
func WithExitCode(code int) Option {
	return func(mp *mockprocessutil.MockProcessConfig) {
		mp.ExitCode = code
	}
}

// NewExecCmd constructs an command executor that can replace standard exec.Cmd
// calls with custom behavior for testing. It takes a series of mock commands
// created with mockprocess.New().
func NewExecCmd(mocks ...*Mock) (func(name string, args ...string) *exec.Cmd, error) {
	mockProcessBinary, err := mockProcessBinaryPath()
	if err != nil {
		return nil, fmt.Errorf("unable to locate mock process binary: %w", err)
	}

	mockProcessMap := map[string]*mockprocessutil.MockProcessConfig{}
	for _, mock := range mocks {
		mockProcessMap[mock.commandRegex] = mock.processConfig
	}

	b, err := json.Marshal(mockProcessMap)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal mockprocessutil.MockProcess map to JSON: %v", err)
	}

	return func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(mockProcessBinary, append([]string{name}, args...)...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", mockprocessutil.EnvHelperMockProcessMap, string(b)))
		return cmd
	}, nil
}

// mockProcessBinaryPath returns the path to the mockprocess binary within
// the current build target's ("go_test") runtime files. The runtime files
// are placed in a different bazel temp on every test run, so the path to the
// binary is deduced from the path of the running test binary (os.Args[0]).
func mockProcessBinaryPath() (string, error) {
	// Returns the file that would have been at the top frame of a stack
	// trace created from this line (this file itself).
	// {buildpacksRepo}/internal/mockprocess/mockprocess.go
	_, callingFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("unable to determine Go runtime information about calling file")
	}

	// {buildpacksRepo}/internal/mockprocess
	callingDir := filepath.Dir(callingFile)

	mockprocessSubPath := "internal/mockprocess"
	// {buildpacksRepo}
	buildpacksRepo := strings.TrimSuffix(filepath.ToSlash(callingDir), mockprocessSubPath)

	// Full path to currently executing test binary
	// {bazelRuntimeRoot}/{buildpacksRepo}/{relativePathToTestBinary}
	executingBinary := filepath.ToSlash(os.Args[0])

	// [{bazelRuntimeRoot}, {relativePathToTestBinary}]
	split := strings.Split(executingBinary, buildpacksRepo)
	if len(split) < 2 {
		return "", fmt.Errorf("unable to determine bazel runtime root, executing test binary: %q, inferred buildpacks repo path: %q, split result: %v", executingBinary, buildpacksRepo, split)
	}

	// {bazelRuntimeRoot}/{buildpacksRepo}/internal/mockprocess/cmd/cmd
	mockProcessBinary := filepath.Join(split[0], buildpacksRepo, "internal", "mockprocess", "cmd", "cmd")
	return filepath.FromSlash(mockProcessBinary), nil
}
