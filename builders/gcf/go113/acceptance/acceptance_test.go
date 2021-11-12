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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	gomod = "google.go.gomod"
)

func vendorSetup(builder, src string) error {
	// The setup function runs `go mod vendor` to vendor dependencies specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod vendor", src, builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vendoring dependencies: %v, output:\n%s", err, out)
	}
	// Vendored functions in Go 1.13 cannot contain go.mod.
	if err := os.Remove(filepath.Join(src, "go.mod")); err != nil {
		return fmt.Errorf("removing go.mod: %v", err)
	}
	return nil
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without deps",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name: "function without framework",
			App:  "no_framework",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:       "vendored function without framwork",
			App:        "no_framework_vendored",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:       "/Func",
			MustOutput: []string{"Found function with vendored dependencies excluding functions-framework"},
		},
		{
			Name: "function with go.sum",
			App:  "no_framework_go_sum",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:       "vendored function with framework",
			App:        "with_framework",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustOutput: []string{"Found function with vendored dependencies including functions-framework"},
			Path:       "/Func",
			Setup:      vendorSetup,
		},
		{
			Name: "function with old framework",
			App:  "with_framework_old_version",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:       "vendored function with old framework",
			App:        "with_framework_old_version",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustOutput: []string{"Found function with vendored dependencies including functions-framework"},
			Path:       "/Func",
			Setup:      vendorSetup,
		},
		{
			Name: "function at /*",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/",
		},
		{
			Name: "function with subdirectories",
			App:  "with_subdir",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name: "declarative http function",
			App:  "declarative_http",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name: "declarative http anonymous function",
			App:  "declarative_anonymous",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name: "declarative and non declarative registration",
			App:  "declarative_old_and_new",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name:                "declarative function signature but wrong target",
			App:                 "declarative_http",
			Env:                 []string{"GOOGLE_FUNCTION_TARGET=ThisDoesntExist"},
			MustMatchStatusCode: 404,
			MustMatch:           "404 page not found",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go113")

			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "no_framework_relative",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustMatch: "the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: func",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go113")

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
