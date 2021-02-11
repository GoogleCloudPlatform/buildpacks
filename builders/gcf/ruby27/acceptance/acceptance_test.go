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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function with dependencies",
			App:  "with_dependencies",
		},
		{
			Name: "function with platform-specific dependencies",
			App:  "with_platform_dependencies",
		},
		{
			Name:   "function with runtime env var",
			App:    "with_env_var",
			RunEnv: []string{"FOO=foo"},
		},
		{
			Name: "function in fn_source file",
			App:  "with_fn_source",
			Env:  []string{"GOOGLE_FUNCTION_SOURCE=sub_dir/custom_file.rb"},
		},
		{
			Name: "function using framework older than 0.7",
			App:  "with_legacy_framework",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=ruby27",
			)
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
			App:       "fail_ruby_version",
			MustMatch: "Could not find gem",
		},
		{
			App:       "fail_framework_missing",
			MustMatch: "unable to execute functions-framework-ruby",
		},
		{
			Name:      "must fail due to missing source file",
			App:       "with_dependencies",
			Env:       []string{"GOOGLE_FUNCTION_SOURCE=missing_file.rb"},
			MustMatch: `GOOGLE_FUNCTION_SOURCE specified file "missing_file.rb" but it does not exist`,
		},
		{
			Name:      "must fail due to incorrect signature",
			App:       "with_dependencies",
			Env:       []string{"GOOGLE_FUNCTION_SIGNATURE_TYPE=cloudevent"},
			MustMatch: `failed to verify function target "testFunction" in source "app.rb": Function "testFunction" does not match type cloudevent`,
		},
		{
			App:       "fail_syntax_error",
			MustMatch: "syntax error",
		},
		{
			App:       "fail_source_missing",
			MustMatch: `expected source file "app.rb" does not exist`,
		},
		{
			App:       "fail_target_missing",
			MustMatch: `failed to verify function target "testFunction" in source "app.rb": Undefined function`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=ruby27",
			)

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
