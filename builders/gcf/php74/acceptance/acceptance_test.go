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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	composer         = "google.php.composer"
	composerGCPBuild = "google.php.composer-gcp-build"
)

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "function without composer.json",
			App:        "no_composer_json",
			MustNotUse: []string{composer, composerGCPBuild},
			MustOutput: []string{
				"Handling function without composer.json",
				"No vendor directory present, installing functions framework",
			},
		},
		{
			Name:       "non default source file",
			App:        "non_default_source_file",
			Env:        []string{"FUNCTION_SOURCE=myfunc.php"},
			MustNotUse: []string{composer, composerGCPBuild},
			MustOutput: []string{
				"Handling function without composer.json",
				"No vendor directory present, installing functions framework",
			},
		},
		{
			Name:       "function without framework dependency",
			App:        "no_framework",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
			MustOutput: []string{"Handling function without dependency on functions framework"},
		},
		{
			Name:       "function with framework dependency",
			App:        "with_framework",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
			MustOutput: []string{"Handling function with dependency on functions framework"},
		},
		{
			Name:       "function with dependencies",
			App:        "with_dependencies",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
			MustOutput: []string{"Handling function without dependency on functions framework"},
		},
		{
			Name:       "function with gcp-build",
			App:        "with_gcp_build",
			MustUse:    []string{composer, composerGCPBuild},
			MustOutput: []string{"Handling function with dependency on functions framework"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env, "FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=php74")

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "fail_syntax_error",
			MustMatch: "Parse error: syntax error",
		},
		{
			App:       "fail_wrong_file",
			MustMatch: "Could not open input file:",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=php74")
			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
