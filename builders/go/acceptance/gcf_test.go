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
package acceptance_test

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

var (
	// goVersionsWithoutGCFSupport contains the list of go versions for which
	// there is GCP or GAE support, but not GCF.
	goVersionsWithoutGCFSupport = []string{"1.12", "1.14", "1.15"}
	// excludedGoVersions is the set of versions for which the regular
	// acceptance tests are not run.
	excludedGoVersions = make(map[string]string)
)

func init() {
	// The tests in this file are lengthy to run due to the number of
	// dependencies pulled in by FF and the test apps. In addition,
	// some of them do not pass for unsupported go versions. For that
	// reason exclude the versions without GCF support.
	for _, v := range goVersionsWithoutGCFSupport {
		excludedGoVersions[v] = v
	}
	// go111 has several differences from the new runtimes and many
	// of the test cases have subtle differences for that reaosn, it
	// is tested seperately.
	excludedGoVersions["1.11"] = "1.11"
}

func vendorSetup(builder, src string) error {
	// The setup function runs `go mod vendor` to vendor dependencies
	// specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod vendor", src, builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vendoring dependencies: %v, output:\n%s", err, out)
	}
	return nil
}

func goSumSetup(builder, src string) error {
	// The setup function runs `go mod vendor` to vendor dependencies
	// specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod tidy", src, builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generating go.sum: %v, output:\n%s", err, out)
	}
	return nil
}

func TestGCFAcceptanceGo(t *testing.T) {
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
			Name:       "vendored function without dependencies",
			App:        "no_framework_vendored",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:       "/Func",
			MustOutput: []string{"Found function with vendored dependencies excluding functions-framework"},
		},
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
			Name:        "declarative cloudevent function",
			App:         "declarative_cloud_event",
			RequestType: acceptance.CloudEventType,
			MustMatch:   "",
			Env:         []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name:        "non declarative cloudevent function",
			App:         "non_declarative_cloud_event",
			RequestType: acceptance.CloudEventType,
			MustMatch:   "",
			Env:         []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_FUNCTION_SIGNATURE_TYPE=cloudevent"},
		},
		{
			Name: "declarative and non declarative registration",
			App:  "declarative_old_and_new",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
		{
			Name:                "no auto registration in main.go if declarative detected",
			App:                 "declarative_cloud_event",
			RequestType:         acceptance.CloudEventType,
			MustMatchStatusCode: 404,
			MustMatch:           "404 page not found",
			// If the buildpack detects the declarative functions package, then
			// functions must be explicitly registered. The main.go written out
			// by the buildpack will NOT use the GOOGLE_FUNCTION_TARGET env var
			// to register a non-declarative function.
			Env: []string{"GOOGLE_FUNCTION_TARGET=NonDeclarativeFunc", "GOOGLE_FUNCTION_SIGNATURE_TYPE=cloudevent"},
		},
		{
			Name:                "declarative function signature but wrong target",
			App:                 "declarative_http",
			Env:                 []string{"GOOGLE_FUNCTION_TARGET=ThisDoesntExist"},
			MustMatchStatusCode: 404,
			MustMatch:           "404 page not found",
		},
		{
			Name:        "background function",
			App:         "background_function",
			RequestType: acceptance.BackgroundEventType,
			Env:         []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_FUNCTION_SIGNATURE_TYPE=event"},
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")
		tc.FilesMustExist = append(tc.FilesMustExist,
			"/layers/google.utils.archive-source/src/source-code.tar.gz",
			"/workspace/.googlebuild/source-code.tar.gz",
		)
		for _, v := range goVersions {
			if shouldSkipVersion(v) {
				continue
			}
			verTC := applyRuntimeVersion(t, tc, v)
			t.Run(verTC.Name, func(t *testing.T) {
				t.Parallel()
				if verTC.Setup != nil {
					t.Skip("TODO: The setup functions require go to be pre-installed which is not true for the unified builder")
				}
				acceptance.TestApp(t, builder, verTC)
			})
		}
	}
}

func shouldSkipVersion(version string) bool {
	_, ok := excludedGoVersions[version]
	return ok
}

func TestGCFFailuresGo(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "no_framework_relative",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustMatch: "the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: func",
		},
		{
			App:       "no_framework",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Setup:     vendorSetup,
			MustMatch: "vendored dependencies must include \"github.com/GoogleCloudPlatform/functions-framework-go\"; if your function does not depend on the module, please add a blank import: `_ \"github.com/GoogleCloudPlatform/functions-framework-go/funcframework\"`",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			if tc.Setup != nil {
				t.Skip("TODO: The setup functions require go to be pre-installed which is not true for the unified builder")
			}

			tc.Env = append(tc.Env,
				"GOOGLE_RUNTIME=go116",
				"GOOGLE_RUNTIME_VERSION=1.16",
				"X_GOOGLE_TARGET_PLATFORM=gcf")

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}

func TestGCFAcceptanceGo111(t *testing.T) {
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
			Name: "vendored function without dependencies",
			App:  "no_framework_vendored",
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
			Name: "function with go.sum",
			App:  "no_framework_go_sum",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:  "vendored function without framework",
			App:   "no_framework",
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
		{
			Name:  "set GOPATH incorrectly",
			App:   "no_framework",
			Env:   []string{"GOOGLE_FUNCTION_TARGET=Func", "GOPATH=/tmp"},
			Path:  "/Func",
			Setup: vendorSetup,
		},
		{
			Name: "X_GOOGLE_ENTRY_POINT ignored",
			App:  "invalid_signature",
			// "Func" is the correct name of target function.
			// X_GOOGLE_ENTRY_POINT is irrelevant for function execution (only
			// used for logging an error message when there's an invalid signature).
			Env:                 []string{"GOOGLE_FUNCTION_TARGET=Func"},
			RunEnv:              []string{"X_GOOGLE_ENTRY_POINT=EntryPoint"},
			MustMatchStatusCode: http.StatusInternalServerError,
			MustMatch:           "func EntryPoint is of the type func(http.ResponseWriter, string), expected func(http.ResponseWriter, *http.Request)",
		},
		{
			Name: "X_GOOGLE_WORKER_PORT used over PORT",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			// "8080" is the correct port to serve on.
			RunEnv: []string{"PORT=1234", "X_GOOGLE_WORKER_PORT=8080"},
		},
		{
			Name: "user module name without dot in path",
			App:  "no_framework_relative",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")
		tc := applyRuntimeVersion(t, tc, "1.11")
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.Setup != nil {
				t.Skip("TODO: The setup functions require go to be pre-installed which is not true for the unified builder")
			}
			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)
			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestGCFFailuresGo111(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App: "with_framework",
			Env: []string{"GOOGLE_FUNCTION_TARGET=Func"},
			// Functions Framework v1.1.0+ supports CloudEvents functions,
			// which requires the cloudevents SDK v2.2.0 which requires Go 1.13+
			MustMatch: "module requires Go 1.13",
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env,
			"X_GOOGLE_TARGET_PLATFORM=gcf",
			"GOOGLE_RUNTIME=go111",
			"GOOGLE_RUNTIME_VERSION=1.11",
		)
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
