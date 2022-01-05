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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktestenv"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

type fakeDetector struct {
	detectFn gcp.DetectFn
}

func (fd *fakeDetector) Detect(ldctx libcnb.DetectContext) (libcnb.DetectResult, error) {
	ctx := gcp.NewContext(gcp.WithApplicationRoot(ldctx.Application.Path), gcp.WithBuildpackRoot(ldctx.Buildpack.Path))
	result, err := fd.detectFn(ctx)
	// detectFn has an interface return type so result may be nil.
	if result == nil {
		return libcnb.DetectResult{}, errors.New("detect did not return a result or an error")
	}
	return result.Result(), err
}

// TestDetect is a helper for testing a buildpack's implementation of /bin/detect.
func TestDetect(t *testing.T, detectFn gcp.DetectFn, testName string, files map[string]string, env []string, want int) {
	TestDetectWithStack(t, detectFn, testName, files, env, "com.stack", want)
}

// TestDetectWithStack is a helper for testing a buildpack's implementation of /bin/detect which allows setting a custom stack name.
func TestDetectWithStack(t *testing.T, detectFn gcp.DetectFn, testName string, files map[string]string, env []string, stack string, want int) {

	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	testArgs := os.Args

	temps, cleanUp := buildpacktestenv.SetUpDetectEnvironmentWithStack(t, stack)
	defer cleanUp()

	for f, c := range files {
		fn := filepath.Join(temps.CodeDir, f)

		if dir := path.Dir(fn); dir != "" {
			if err := os.MkdirAll(dir, 0744); err != nil {
				t.Fatalf("creating directory tree %s: %v", dir, err)
			}
		}

		if err := ioutil.WriteFile(fn, []byte(c), 0644); err != nil {
			t.Fatalf("writing file %s: %v", fn, err)
		}
	}

	ctx := gcp.NewContext(gcp.WithApplicationRoot(temps.CodeDir), gcp.WithBuildpackRoot(temps.BuildpackDir))

	// Invoke detect in a separate process.
	// Otherwise, detect could exit and stop the test.
	if os.Getenv("TEST_DETECT_EXITING") == "1" {
		libcnb.Detect(&fakeDetector{detectFn: detectFn})
	} else {
		cmd := exec.Command(filepath.Join(testDir, testArgs[0]), fmt.Sprintf("-test.run=TestDetect/^%s$", strings.ReplaceAll(testName, " ", "_")))
		cmd.Env = append(os.Environ(), "TEST_DETECT_EXITING=1")
		cmd.Dir = ctx.ApplicationRoot()

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
