// Copyright 2021 Google LLC
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

// Package buildpacktestenv contains utilities for setting up environments
// for buildpack tests.
package buildpacktestenv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	// EnvHelperMockProcessMap is the env var used to communicate the intended
	// behavior of the mock process for various commands. It contains a
	// map[string]MockProcess serialized to JSON.
	EnvHelperMockProcessMap = "HELPER_MOCK_PROCESS_MAP"
)

// TempDirs represents temp directories used for buildpack environment setup.
type TempDirs struct {
	LayersDir    string
	PlatformDir  string
	CodeDir      string
	BuildpackDir string
	PlanFile     string
}

// TempWorkingDir creates a temp dir, sets the current working directory to it, and returns a clean up function to restore everything back.
func TempWorkingDir(t *testing.T) (string, func()) {
	t.Helper()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	newwd, err := ioutil.TempDir("", "source-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	if err := os.Chdir(newwd); err != nil {
		t.Fatalf("setting current dir to %q: %v", newwd, err)
	}

	return newwd, func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restoring old current dir to %q: %v", oldwd, err)
		}
		if err := os.RemoveAll(newwd); err != nil {
			t.Fatalf("deleting temp dir %q: %v", newwd, err)
		}
	}
}

// SetUpTempDirs sets up temp directories that mimic the layers of buildpacks.
func SetUpTempDirs(t *testing.T, customCodeDir string) TempDirs {
	t.Helper()
	layersDir := t.TempDir()
	platformDir := t.TempDir()
	codeDir := ""
	if customCodeDir != "" {
		err := os.Mkdir(filepath.Join(os.TempDir(), customCodeDir), 0700)
		if err != nil {
			t.Fatalf("creating code dir: %v", err)
		}
		codeDir = filepath.Join(os.TempDir(), customCodeDir)
	} else {
		codeDir = t.TempDir()
	}
	buildpackDir := t.TempDir()

	stack := "com.stack"
	buildpackTOML := fmt.Sprintf(`
api = "0.9"

[buildpack]
id = "my-id"
version = "my-version"
name = "my-name"

[[stacks]]
id = "%s"
`, stack)

	if err := ioutil.WriteFile(filepath.Join(buildpackDir, "buildpack.toml"), []byte(buildpackTOML), 0644); err != nil {
		t.Fatalf("writing buildpack.toml: %v", err)
	}

	planTOML := `
[[entries]]
name = "entry-name"
version = "entry-version"
[entries.metadata]
  entry-meta-key = "entry-meta-value"
`
	if err := ioutil.WriteFile(filepath.Join(buildpackDir, "plan.toml"), []byte(planTOML), 0644); err != nil {
		t.Fatalf("writing plan.toml: %v", err)
	}

	if err := os.Setenv("CNB_STACK_ID", stack); err != nil {
		t.Fatalf("setting env var CNB_STACK_ID: %v", err)
	}

	if err := os.Setenv("CNB_BUILDPACK_DIR", buildpackDir); err != nil {
		t.Fatalf("setting env var CNB_BUILDPACK_DIR: %v", err)
	}

	// Set Plartform dir
	if err := os.Setenv("CNB_PLATFORM_DIR", platformDir); err != nil {
		t.Fatalf("setting env var CNB_PLATFORM_DIR: %v", err)
	}

	// Set Build plan path
	if err := os.Setenv("CNB_BUILD_PLAN_PATH", filepath.Join(buildpackDir, "plan.toml")); err != nil {
		t.Fatalf("setting env var CNB_BUILD_PLAN_PATH: %v", err)
	}

	// Set Layers dir
	if err := os.Setenv("CNB_LAYERS_DIR", layersDir); err != nil {
		t.Fatalf("setting env var CNB_LAYERS_DIR: %v", err)
	}

	// Set CNB_BP_PLAN_PATH
	if err := os.Setenv("CNB_BP_PLAN_PATH", filepath.Join(buildpackDir, "plan.toml")); err != nil {
		t.Fatalf("setting env var CNB_BP_PLAN_PATH: %v", err)
	}

	temps := TempDirs{
		CodeDir:      codeDir,
		LayersDir:    layersDir,
		PlatformDir:  platformDir,
		BuildpackDir: buildpackDir,
		PlanFile:     filepath.Join(buildpackDir, "plan.toml"),
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(codeDir); err != nil {
			t.Fatalf("removing code dir %q: %v", codeDir, err)
		}
		if err := os.Unsetenv("CNB_STACK_ID"); err != nil {
			t.Fatalf("unsetting CNB_STACK_ID: %v", err)
		}
		if err := os.Unsetenv("CNB_BUILDPACK_DIR"); err != nil {
			t.Fatalf("unsetting CNB_BUILDPACK_DIR: %v", err)
		}
		if err := os.Unsetenv("CNB_PLATFORM_DIR"); err != nil {
			t.Fatalf("unsetting CNB_PLATFORM_DIR: %v", err)
		}
		if err := os.Unsetenv("CNB_BUILD_PLAN_PATH"); err != nil {
			t.Fatalf("unsetting CNB_BUILD_PLAN_PATH: %v", err)
		}
	})
	return temps
}

// MockProcess encapsulates the behavior of a mock process for test.
// To add more behaviors to the mock process, expand this struct and implement
// the corresponding handling in
// internal/buildpacktest/mockprocess/mockprocess.go
type MockProcess struct {
	// Stdout is the message that should be printed to stdout.
	Stdout string
	// Stderr is the message that should be printed to stderr.
	Stderr string
	// ExitCode is the exit code that the process should use.
	ExitCode int
}

// NewMockExecCmd constructs an ExecCmd that can replace standard exec.Cmd calls
// with custom behavior for testing. It takes the path to the mock process
// binary as the first argument, and a map of
// { command : mock action to simulate the commands behavior }. The command
// must be at least a partial match to the full command (binary name and all
// args) that would have been executed by exec.Cmd.
func NewMockExecCmd(t *testing.T, mockProcess string, commands map[string]*MockProcess) func(name string, args ...string) *exec.Cmd {
	t.Helper()

	b, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("unable to marshal MockProcess map to JSON: %v", err)
	}

	return func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(mockProcess, append([]string{name}, args...)...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvHelperMockProcessMap, string(b)))
		return cmd
	}
}

// UnmarshalMockProcessMap is a utility function that marshals a
// map[string]MockProcess from JSON.
func UnmarshalMockProcessMap(data string) (map[string]*MockProcess, error) {
	var mocks map[string]*MockProcess
	if err := json.Unmarshal([]byte(data), &mocks); err != nil {
		return mocks, err
	}

	return mocks, nil
}
