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
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:          "function without framework",
			App:           "without_framework",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies", "WARNING: You are using pip version"},
		},
		{
			Name:            "function with dependencies",
			App:             "with_dependencies",
			EnableCacheTest: true,
			// No MustNotOutput WARNING because hatch non-deterministically produces an incompatible dependency tree.
		},
		{
			Name:          "function with framework",
			App:           "with_framework",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
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
			// No MustNotOutput WARNING because spacy non-deterministically produces an incompatible dependency tree.
		},
		{
			Name:          "function with runtime env var",
			App:           "with_env_var",
			RunEnv:        []string{"FOO=foo"},
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
		},
		{
			Name:          "function returns None",
			App:           "returns_none",
			MustMatch:     "OK",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
		},
		{
			Name:          "function with env var ENTRY_POINT",
			App:           "env_var_entry_point",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
		},
		{
			Name:          "function with compat dependencies with framework",
			App:           "compat_dependencies_with_framework",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
		},
		{
			Name:          "function with compat dependencies without framework",
			App:           "compat_dependencies_without_framework",
			MustNotOutput: []string{"WARNING: Found incompatible dependencies"},
		},
		{
			Name: "function with conflicting dependencies",
			App:  "conflicting_dependencies",
			// No MustNotOutput WARNING because transitive dependencies in the test function are not pinned.
		},
		{
			Name:       "allow broken dependencies",
			App:        "fail_broken_dependencies",
			MustOutput: []string{`WARNING: Found incompatible dependencies: "functions-framework 3.0.0 has requirement flask<3.0,>=1.0, but you have flask 0.12.5."`},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=python37",
				"X_GOOGLE_TARGET_PLATFORM=gcf",
			)
			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App: "fail_syntax_error",
			Env: []string{
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=python37",
				"X_GOOGLE_TARGET_PLATFORM=gcf",
			},
			MustMatch: "SyntaxError: invalid syntax",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
