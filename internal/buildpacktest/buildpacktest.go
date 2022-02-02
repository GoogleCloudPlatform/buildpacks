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

// Package buildpacktest contains utilities for testing buildpacks that
// use the `gcpbuildpack` package.
package buildpacktest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktestenv"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

type buildpackPhase string

const (
	detectPhase buildpackPhase = "Detect"
	buildPhase  buildpackPhase = "Build"

	// runTestAsHelperProcessEnv is an env variable that signals the current
	// golang test being run is actually a child process of the main golang
	// test process. The child process is used to execute the buildpack phase
	// under test without impacting the main test process. The env value is
	// the buildpackPhase to execute.
	//
	// This is similar to how the exec package tests exec.Command
	// (see https://golang.org/src/os/exec/exec_test.go).
	runTestAsHelperProcessEnv = "RUN_TEST_AS_HELPER_PROCESS"
)

type config struct {
	buildpackPhase buildpackPhase
	buildFn        gcp.BuildFn
	detectFn       gcp.DetectFn
	testName       string
	files          map[string]string
	envs           []string
	stack          string
	want           int
}

// Result encapsulates the result of a buildpack phase.
type Result struct {
	Stdout string
	Stderr string
}

// TestDetect is a helper for testing a buildpack's implementation of /bin/detect.
// This MUST be called from a test function with the name `func TestDetect(t *testing.T)`
// A child process will be started that looks for that test name. The child
// process will run a buildpack phase instead of the test again, however.
func TestDetect(t *testing.T, detectFn gcp.DetectFn, testName string, files map[string]string, envs []string, want int) {
	TestDetectWithStack(t, detectFn, testName, files, envs, "com.stack", want)
}

// TestDetectWithStack is a helper for testing a buildpack's implementation of /bin/detect which allows setting a custom stack name.
// This MUST be called from a test function with the stub `func TestDetectWithStack(t *testing.T)`
// A child process will be started that looks for that test name. The child
// process will run a buildpack phase instead of the test again, however.
func TestDetectWithStack(t *testing.T, detectFn gcp.DetectFn, testName string, files map[string]string, envs []string, stack string, want int) {
	result, err := runBuildpackPhaseForTest(t, &config{
		buildpackPhase: detectPhase,
		detectFn:       detectFn,
		testName:       testName,
		files:          files,
		envs:           envs,
		stack:          stack,
		want:           want,
	})

	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() != want {
		t.Errorf("unexpected exit status %d, want %d", e.ExitCode(), want)
		t.Errorf("\nStdout: %s\nStderr: %s", result.Stdout, result.Stderr)
	}

	if err == nil && want != 0 {
		t.Errorf("unexpected exit status 0, want %d", want)
		t.Errorf("\nStdout: %s\nStderr: %s", result.Stdout, result.Stderr)
	}
}

// runBuildpackPhaseForTest runs a buildpack phase as a separate child process.
// A child process is used to avoid the test suite itself being terminated by
// errant calls to os.Exit() in the buildpack.
func runBuildpackPhaseForTest(t *testing.T, cfg *config) (*Result, error) {
	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	if bp := os.Getenv(runTestAsHelperProcessEnv); bp != "" {
		runBuildpackPhaseMain(t, cfg)
	} else {
		// Invoke buildpack phase in a separate process. This is done
		// by executing the current tests again in a separate process and adding
		// the env var that signals the buildpack phase should be run (args[0]
		// is the current running binary).
		testBinary := filepath.Join(testDir, os.Args[0])
		cmd := exec.Command(testBinary, fmt.Sprintf("-test.run=Test%s/^%s$", cfg.buildpackPhase, strings.ReplaceAll(cfg.testName, " ", "_")))
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", runTestAsHelperProcessEnv, cfg.buildpackPhase))

		for _, e := range cfg.envs {
			cmd.Env = append(cmd.Env, e)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		t.Logf("running command %v", cmd)

		err := cmd.Run()
		result := &Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}

		return result, err
	}

	return &Result{}, nil
}

// runBuildpackPhaseMain runs a buildpack phase. It is the equivalent
// of `func main()` for a helper process. To avoid confusion, it is written
// like the main of a standard Go app, using "log.Fatalf" in place of
// "t.Fatalf".
func runBuildpackPhaseMain(t *testing.T, cfg *config) {
	temps := buildpacktestenv.SetUpTempDirs(t)
	opts := []gcp.ContextOption{gcp.WithApplicationRoot(temps.CodeDir), gcp.WithBuildpackRoot(temps.BuildpackDir)}
	ctx := gcp.NewContext(opts...)

	for f, c := range cfg.files {
		fn := filepath.Join(temps.CodeDir, f)

		if dir := path.Dir(fn); dir != "" {
			if err := os.MkdirAll(dir, 0744); err != nil {
				log.Fatalf("creating directory tree %s: %v", dir, err)
			}
		}

		if err := ioutil.WriteFile(fn, []byte(c), 0644); err != nil {
			log.Fatalf("writing file %s: %v", fn, err)
		}
	}

	oldDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(temps.CodeDir); err != nil {
		log.Fatalf("changing to code dir %q: %v", temps.CodeDir, err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			log.Fatalf("changing back to old working directory %q: %v", oldDir, err)
		}
	}()

	if cfg.buildpackPhase == buildPhase {
		if err := cfg.buildFn(ctx); err != nil {
			log.Fatalf("build error: %v", err)
		}
	} else {
		detect, err := cfg.detectFn(ctx)
		if err != nil {
			log.Fatalf("detect error: %v", err)
		}

		// Mimics the exit code of libcnb library when the detect function
		// succeeds but does not pass detect.
		if !detect.Result().Pass {
			os.Exit(100)
		}
	}
}
