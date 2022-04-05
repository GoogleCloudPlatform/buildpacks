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

package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
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
			Env:        []string{"GOOGLE_FUNCTION_SOURCE=myfunc.php"},
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
		{
			Name:       "function with vendor dir but no framework",
			App:        "vendored_no_framework",
			MustNotUse: []string{composer, composerGCPBuild},
			MustOutput: []string{
				"Handling function without composer.json",
				"Functions framework is not present at vendor/google/cloud-functions-framework",
				// The version spec of the functions framework follows this string.
				// Omitting it here so we don't fail when it's upgraded.
				"Installing functions framework google/cloud-functions-framework:",
			},
		},
		{
			Name: "declarative http function",
			App:  "declarative_http",
		},
		{
			Name:        "declarative cloudevent function",
			App:         "declarative_cloud_event",
			RequestType: acceptance.CloudEventType,
		},
		{
			Name:        "non declarative cloudevent function",
			Env:         []string{"GOOGLE_FUNCTION_SIGNATURE_TYPE=cloudevent"},
			App:         "non_declarative_cloud_event",
			RequestType: acceptance.CloudEventType,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env, "GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=php74", "X_GOOGLE_TARGET_PLATFORM=gcf")
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
			App:       "fail_syntax_error",
			MustMatch: "Parse error: syntax error",
		},
		{
			App:       "fail_vendored_framework_no_router_script",
			MustMatch: `functions framework router script vendor/bin/router\.php is not present`,
		},
		{
			App:       "fail_vendored_no_framework_no_installed_json",
			MustMatch: `vendor/composer/installed\.json is not present, so it appears that Composer was not used to install dependencies\.`,
		},
		{
			App:       "fail_wrong_file",
			MustMatch: "Could not open input file:",
		},
		{
			// todo(mtraver): This acceptance test must exist in the OSS buildpack for PHP when created per b/156265858.
			App:       "no_composer_json",
			Env:       []string{"GOOGLE_FUNCTION_SOURCE=file_does_not_exist.php"},
			MustMatch: "Could not open input file: file_does_not_exist.php",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=php74", "X_GOOGLE_TARGET_PLATFORM=gcf")
			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
