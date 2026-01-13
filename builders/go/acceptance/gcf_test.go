// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)
	testCases := []acceptance.Test{
		{
			VersionInclusionConstraint: ">= 1.14.0",
			Name:                       "function without deps",
			App:                        "no_deps",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:                       "/Func",
		},
		{
			VersionInclusionConstraint: ">= 1.14.0",
			Name:                       "vendored function without dependencies",
			App:                        "no_framework_vendored_no_go_mod",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:                       "/Func",
			MustOutput:                 []string{"Found function with vendored dependencies excluding functions-framework"},
			Labels: map[string]string{
				"google.functions-framework-version": `{"runtime":"go","version":"v1.9.2","injected":true}`,
			},
		},
		{
			VersionInclusionConstraint: ">= 1.17.0",
			Name:                       "function without framework",
			App:                        "no_framework",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:                       "/Func",
			MustOutput:                 []string{"go.sum not found, generating"},
			EnableCacheTest:            true,
			Labels: map[string]string{
				"google.functions-framework-version": `{"runtime":"go","version":"v1.9.2","injected":true}`,
			},
		},
		{
			VersionInclusionConstraint: ">= 1.17.0",
			Name:                       "function with go.sum",
			App:                        "no_framework_go_sum",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:                       "/Func",
			MustNotOutput:              []string{"go.sum not found, generating"},
			Labels: map[string]string{
				"google.functions-framework-version": `{"runtime":"go","version":"v1.9.2","injected":true}`,
			},
		},
		{
			VersionInclusionConstraint: ">= 1.17.0",
			Name:                       "function at /*",
			App:                        "no_framework",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path:                       "/",
		},
		{
			VersionInclusionConstraint: ">= 14.0.0",
			Name:                       "function with subdirectories",
			App:                        "with_subdir",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func"},
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
			VersionInclusionConstraint: ">= 1.14.0",
			Name:                       "non declarative cloudevent function",
			App:                        "non_declarative_cloud_event",
			RequestType:                acceptance.CloudEventType,
			MustMatch:                  "",
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_FUNCTION_SIGNATURE_TYPE=cloudevent"},
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
			VersionInclusionConstraint: ">= 1.14.0",
			Name:                       "background function",
			App:                        "background_function",
			RequestType:                acceptance.BackgroundEventType,
			Env:                        []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_FUNCTION_SIGNATURE_TYPE=event"},
		},
		{
			Name: "function with versioned module",
			App:  "with_versioned_mod",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Labels: map[string]string{
				"google.functions-framework-version": `{"runtime":"go","version":"v1.6.1","injected":false}`,
			},
		},
		{
			Name:                "function with go.mod using replace directive",
			App:                 "declarative_with_replace",
			Env:                 []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustMatchStatusCode: 200,
			MustMatch:           "PASS",
			Labels: map[string]string{
				"google.functions-framework-version": `{"runtime":"go","version":"v1.7.0","injected":false}`,
			},
		},
	}
	if !acceptance.ShouldTestVersion(t, "1.13") {
		testCases = append(testCases,
			acceptance.Test{
				Name: "with framework go mod vendored",
				App:  "with_framework_vendored",
				Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
				Labels: map[string]string{
					"google.functions-framework-version": `{"runtime":"go","version":"v1.6.1","injected":false}`,
				},
			},
		)
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")
		tc.FilesMustExist = append(tc.FilesMustExist,
			"/layers/google.utils.archive-source/src/source-code.tar.gz",
			"/workspace/.googlebuild/source-code.tar.gz",
		)
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)
	testCases := []acceptance.FailureTest{
		{
			Name:      "no dot in mod name",
			App:       "no_dot_in_mod_name",
			MustMatch: "the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: func",
		},
		{
			Name:      "go mod and vendor no framework",
			App:       "without_framework_vendored",
			MustMatch: "vendored dependencies must include \"github.com/GoogleCloudPlatform/functions-framework-go\"; if your function does not depend on the module, please add a blank import: `_ \"github.com/GoogleCloudPlatform/functions-framework-go/funcframework\"`",
		},
		{
			Name:      "vendored function without dependencies or injection",
			App:       "no_framework_vendored_no_go_mod",
			Env:       []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=True"},
			MustMatch: "skipping automatic framework injection has been enabled",
		},
		{
			Name:      "go mod function without dependencies or injection",
			App:       "no_framework",
			Env:       []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=True"},
			MustMatch: "skipping automatic framework injection has been enabled",
		},
	}
	if !acceptance.ShouldTestVersion(t, "1.13") {
		testCases = append(testCases,
			acceptance.FailureTest{
				Name:      "without framework go mod vendored",
				App:       "without_framework_vendored",
				MustMatch: "vendored dependencies must include \"github.com/GoogleCloudPlatform/functions-framework-go\"; if your function does not depend on the module, please add a blank import: `_ \"github.com/GoogleCloudPlatform/functions-framework-go/funcframework\"`",
			})
	}
	for _, tc := range testCases {
		tc.Env = append(tc.Env,
			"GOOGLE_FUNCTION_TARGET=Func",
			"X_GOOGLE_TARGET_PLATFORM=gcf",
		)
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
