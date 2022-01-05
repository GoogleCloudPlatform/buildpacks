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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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

func setOSArgs(t *testing.T, args []string) func() {
	t.Helper()
	oldArgs := os.Args
	os.Args = args
	return func() {
		os.Args = oldArgs
	}
}

func setUpTempDirs(t *testing.T, stack string) (TempDirs, func()) {
	t.Helper()
	LayersDir, err := ioutil.TempDir("", "layers-")
	if err != nil {
		t.Fatalf("creating layers dir: %v", err)
	}
	PlatformDir, err := ioutil.TempDir("", "platform-")
	if err != nil {
		t.Fatalf("creating platform dir: %v", err)
	}
	CodeDir, err := ioutil.TempDir("", "CodeDir-")
	if err != nil {
		t.Fatalf("creating code dir: %v", err)
	}
	BuildpackDir, err := ioutil.TempDir("", "buildpack-")
	if err != nil {
		t.Fatalf("creating buildpack dir: %v", err)
	}

	// set up cwd
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(CodeDir); err != nil {
		t.Fatalf("changing to code dir %q: %v", CodeDir, err)
	}

	buildpackTOML := fmt.Sprintf(`
api = "0.5"

[buildpack]
id = "my-id"
version = "my-version"
name = "my-name"

[[stacks]]
id = "%s"
`, stack)

	if err := ioutil.WriteFile(filepath.Join(BuildpackDir, "buildpack.toml"), []byte(buildpackTOML), 0644); err != nil {
		t.Fatalf("writing buildpack.toml: %v", err)
	}

	planTOML := `
[[entries]]
name = "entry-name"
version = "entry-version"
[entries.metadata]
  entry-meta-key = "entry-meta-value"
`
	if err := ioutil.WriteFile(filepath.Join(BuildpackDir, "plan.toml"), []byte(planTOML), 0644); err != nil {
		t.Fatalf("writing plan.toml: %v", err)
	}

	if err := os.Setenv("CNB_STACK_ID", stack); err != nil {
		t.Fatalf("setting env var CNB_STACK_ID: %v", err)
	}

	temps := TempDirs{
		CodeDir:      CodeDir,
		LayersDir:    LayersDir,
		PlatformDir:  PlatformDir,
		BuildpackDir: BuildpackDir,
		PlanFile:     filepath.Join(BuildpackDir, "plan.toml"),
	}

	return temps, func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("changing back to old working directory %q: %v", oldDir, err)
		}
		if err := os.RemoveAll(CodeDir); err != nil {
			t.Fatalf("removing code dir %q: %v", CodeDir, err)
		}
		if err := os.RemoveAll(PlatformDir); err != nil {
			t.Fatalf("removing platform dir %q: %v", PlatformDir, err)
		}
		if err := os.RemoveAll(LayersDir); err != nil {
			t.Fatalf("removing layers dir %q: %v", LayersDir, err)
		}
		if err := os.RemoveAll(BuildpackDir); err != nil {
			t.Fatalf("removing buildpac dir %q: %v", BuildpackDir, err)
		}
		if err := os.Unsetenv("CNB_STACK_ID"); err != nil {
			t.Fatalf("unsetting CNB_STACK_ID: %v", err)
		}
	}
}

// SetUpDetectEnvironment sets up an environment for testing buildpack detect
// functionality.
func SetUpDetectEnvironment(t *testing.T) (TempDirs, func()) {
	return SetUpDetectEnvironmentWithStack(t, "com.stack")
}

// SetUpDetectEnvironmentWithStack sets up an environment for testing buildpack detect
// functionality with a custom stack name.
func SetUpDetectEnvironmentWithStack(t *testing.T, stack string) (TempDirs, func()) {
	t.Helper()
	temps, cleanUpTempDirs := setUpTempDirs(t, stack)
	cleanUpArgs := setOSArgs(t, []string{filepath.Join(temps.BuildpackDir, "bin", "detect"), temps.PlatformDir, temps.PlanFile})

	return temps, func() {
		cleanUpArgs()
		cleanUpTempDirs()
	}
}

// SetUpBuildEnvironment sets up an environment for testing buildpack buil
// functionality.
func SetUpBuildEnvironment(t *testing.T) (TempDirs, func()) {
	t.Helper()
	temps, cleanUpTempDirs := setUpTempDirs(t, "com.stack")
	cleanUpArgs := setOSArgs(t, []string{filepath.Join(temps.BuildpackDir, "bin", "build"), temps.LayersDir, temps.PlatformDir, temps.PlanFile})

	return temps, func() {
		cleanUpArgs()
		cleanUpTempDirs()
	}
}
