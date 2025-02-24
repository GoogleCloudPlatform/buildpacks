// Copyright 2020 Google LLC
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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:          "function without framework",
			App:           "without_framework",
			MustNotOutput: []string{`WARNING: You are using pip version`},
		},
		{
			Name:          "function without framework and allow injection",
			App:           "without_framework",
			Env:           []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=False"},
			MustNotOutput: []string{`WARNING: You are using pip version`},
		},
		{
			Name:            "function with dependencies",
			App:             "with_dependencies",
			EnableCacheTest: true,
		},
		{
			Name: "function with framework",
			App:  "with_framework",
		},
		{
			Name: "function using http declarative function signatures",
			App:  "use_declarative",
		},
		{
			Name:        "function using CloudEvent declarative function signatures",
			App:         "use_cloud_event_declarative",
			MustMatch:   "OK",
			RequestType: acceptance.CloudEventType,
		},
		{
			Name: "function with framework and dependency bin",
			App:  "with_framework_bin_conflict",
			// TODO(harisam): Remove this constraint once spacy support is added for python 3.13.
			VersionInclusionConstraint: "< 3.13.0",
		},
		{
			Name:   "function with runtime env var",
			App:    "with_env_var",
			RunEnv: []string{"FOO=foo"},
		},
		{
			Name: "function has right number of dependencies",
			App:  "list_dependencies",
			// The list_dependencies app has a dependency on the libexpat OS package which isn't installed
			// in the min run image.
			SkipStacks: []string{"google.min.22"},
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := applyStaticTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	tc.Path = "/testFunction"
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	tc.FilesMustExist = append(tc.FilesMustExist,
		"/layers/google.utils.archive-source/src/source-code.tar.gz",
		"/workspace/.googlebuild/source-code.tar.gz",
	)
	return tc
}

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "fail_syntax_error",
			MustMatch: "SyntaxError: invalid syntax",
		},
		{
			App:       "fail_broken_dependencies",
			MustMatch: `functions-framework .* has requirement flask.*,>=.*, but you have flask 0\.12\.5`,
			// this is only a warning in python37
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:      "function without dependencies or injection",
			App:       "without_framework",
			Env:       []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=True"},
			MustMatch: "skipping automatic framework injection has been enabled",
		},
		{
			Name:      "use pip vendored deps - framework not vendored",
			App:       "without_framework",
			Env:       []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=vendor"},
			MustMatch: "Vendored dependencies detected, please add functions-framework to requirements.txt and download it using pip",
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := applyStaticFailureTestOptions(tc)
		t.Run(tc.App, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}

func applyStaticFailureTestOptions(tc acceptance.FailureTest) acceptance.FailureTest {
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	return tc
}
