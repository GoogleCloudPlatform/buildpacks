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
package acceptance

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func vendorSetup(builder, src string) error {
	// The setup function runs `go mod vendor` to vendor dependencies specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod vendor", src, builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vendoring dependencies: %v, output:\n%s", err, out)
	}
	return nil
}

func goSumSetup(builder, src string) error {
	// The setup function runs `go mod vendor` to vendor dependencies specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod tidy", src, builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generating go.sum: %v, output:\n%s", err, out)
	}
	return nil
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "function without framework",
			App:        "no_framework",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:       "/Func",
			MustOutput: []string{"go.sum not found, generating"},
		},
		{
			Name:          "function with go.sum",
			App:           "no_framework",
			Env:           []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Setup:         goSumSetup,
			Path:          "/Func",
			MustNotOutput: []string{"go.sum not found, generating"},
		},
		{
			Name:  "vendored function without framework",
			App:   "no_framework",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name:  "vendored function with framework",
			App:   "with_framework",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "function with old framework",
			App:  "with_framework_old_version",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:  "vendored function with old framework",
			App:   "with_framework_old_version",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "function at /*",
			App:  "no_framework",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/",
		},
		{
			Name: "function with subdirectories",
			App:  "with_subdir",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go116")

			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	// TODO(b/181910032): Remove when stack images are published.
	if acceptance.PullImages() {
		t.Skip("Disabled for continuous builds")
	}

	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "no_framework_relative",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustMatch: "the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: func",
		},
		{
			App:       "no_deps",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustMatch: "function build requires go.mod file",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go116")

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
