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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpack/libbuildpack/buildpack"
)

type tempDirs struct {
	layersDir    string
	platformDir  string
	codeDir      string
	buildpackDir string
	planFile     string
}

// TestDetect is a helper for testing a buildpack's implementation of /bin/detect.
func TestDetect(t *testing.T, detectFn DetectFn, testName string, files map[string]string, env []string, want int) {
	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	testArgs := os.Args

	temps, cleanUp := setUpDetectEnvironment(t)
	defer cleanUp()

	for f, c := range files {
		fn := filepath.Join(temps.codeDir, f)
		if err := ioutil.WriteFile(fn, []byte(c), 0644); err != nil {
			t.Fatalf("writing file %s: %v", fn, err)
		}
	}

	ctx := newDetectContext()
	ctx.applicationRoot = temps.codeDir
	ctx.buildpackRoot = temps.buildpackDir

	// Invoke detect in a separate process.
	// Otherwise, detect could exit and stop the test.
	if os.Getenv("TEST_DETECT_EXITING") == "1" {
		detect(detectFn)
	} else {
		cmd := exec.Command(filepath.Join(testDir, testArgs[0]), fmt.Sprintf("-test.run=TestDetect/^%s$", strings.ReplaceAll(testName, " ", "_")))
		cmd.Env = append(os.Environ(), "TEST_DETECT_EXITING=1")

		for _, e := range env {
			cmd.Env = append(cmd.Env, e)
		}

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		t.Logf("running command %v", cmd)

		err = cmd.Run()
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != want {
			t.Errorf("unexpected exit status %d, want %d", e.ExitCode(), want)
			t.Errorf("\n%s", out.String())
		}

		if err == nil && want != 0 {
			t.Errorf("unexpected exit status 0, want %d", want)
			t.Errorf("\n%s", out.String())
		}
	}
}

// tempWorkingDir creates a temp dir, sets the current working directory to it, and returns a clean up function to restore everything back.
func tempWorkingDir(t *testing.T) (string, func()) {
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

func simpleContext(t *testing.T) (*Context, func()) {
	t.Helper()
	_, cleanUp := setUpDetectEnvironment(t)
	c := NewContext(buildpack.Info{ID: "my-id", Version: "my-version", Name: "my-name"})
	return c, cleanUp
}

func setOSArgs(t *testing.T, args []string) func() {
	t.Helper()
	oldArgs := os.Args
	os.Args = args
	return func() {
		os.Args = oldArgs
	}
}

func setUpTempDirs(t *testing.T) (tempDirs, func()) {
	t.Helper()
	layersDir, err := ioutil.TempDir("", "layers-")
	if err != nil {
		t.Fatalf("creating layers dir: %v", err)
	}
	platformDir, err := ioutil.TempDir("", "platform-")
	if err != nil {
		t.Fatalf("creating platform dir: %v", err)
	}
	codeDir, err := ioutil.TempDir("", "codedir-")
	if err != nil {
		t.Fatalf("creating code dir: %v", err)
	}
	buildpackDir, err := ioutil.TempDir("", "buildpack-")
	if err != nil {
		t.Fatalf("creating buildpack dir: %v", err)
	}

	// set up cwd
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(codeDir); err != nil {
		t.Fatalf("changing to code dir %q: %v", codeDir, err)
	}

	buildpackTOML := `
[buildpack]
id = "my-id"
version = "my-version"
name = "my-name"

[[stacks]]
id = "com.stack"
`
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

	if err := os.Setenv("CNB_STACK_ID", "com.stack"); err != nil {
		t.Fatalf("setting env var CNB_STACK_ID: %v", err)
	}

	temps := tempDirs{
		codeDir:      codeDir,
		layersDir:    layersDir,
		platformDir:  platformDir,
		buildpackDir: buildpackDir,
		planFile:     filepath.Join(buildpackDir, "plan.toml"),
	}

	return temps, func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("changing back to old working directory %q: %v", oldDir, err)
		}
		if err := os.RemoveAll(codeDir); err != nil {
			t.Fatalf("removing code dir %q: %v", codeDir, err)
		}
		if err := os.RemoveAll(platformDir); err != nil {
			t.Fatalf("removing platform dir %q: %v", platformDir, err)
		}
		if err := os.RemoveAll(layersDir); err != nil {
			t.Fatalf("removing layers dir %q: %v", layersDir, err)
		}
		if err := os.RemoveAll(buildpackDir); err != nil {
			t.Fatalf("removing buildpac dir %q: %v", buildpackDir, err)
		}
		if err := os.Unsetenv("CNB_STACK_ID"); err != nil {
			t.Fatalf("unsetting CNB_STACK_ID: %v", err)
		}
	}
}

func setUpDetectEnvironment(t *testing.T) (tempDirs, func()) {
	t.Helper()
	temps, cleanUpTempDirs := setUpTempDirs(t)
	cleanUpArgs := setOSArgs(t, []string{filepath.Join(temps.buildpackDir, "bin", "detect"), temps.platformDir, temps.planFile})

	return temps, func() {
		cleanUpArgs()
		cleanUpTempDirs()
	}
}

func setUpBuildEnvironment(t *testing.T) (tempDirs, func()) {
	t.Helper()
	temps, cleanUpTempDirs := setUpTempDirs(t)
	cleanUpArgs := setOSArgs(t, []string{filepath.Join(temps.buildpackDir, "bin", "build"), temps.layersDir, temps.platformDir, temps.planFile})

	return temps, func() {
		cleanUpArgs()
		cleanUpTempDirs()
	}
}
