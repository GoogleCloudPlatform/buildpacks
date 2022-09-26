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
		},
		{
			Name:   "function with runtime env var",
			App:    "with_env_var",
			RunEnv: []string{"FOO=foo"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=python39",
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
			App:       "fail_syntax_error",
			MustMatch: "SyntaxError: invalid syntax",
		},
		{
			App:       "fail_broken_dependencies",
			MustMatch: `functions-framework .* has requirement flask<3\.0,>=1\.0, but you have flask 0\.12\.5`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		tc.Env = append(tc.Env,
			"GOOGLE_FUNCTION_TARGET=testFunction",
			"GOOGLE_RUNTIME=python39",
			"X_GOOGLE_TARGET_PLATFORM=gcf",
		)
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
