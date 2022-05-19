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

const (
	npm  = "google.nodejs.npm"
	yarn = "google.nodejs.yarn"

	// Firebase functions expect FIREBASE_CONFIG & GCLOUD_PROJECT env vars at run time.
	// Otherwise, initializing the Firebase Admin SDK would fail.
	gcloudProject  = "GCLOUD_PROJECT=foo"
	firebaseConfig = `FIREBASE_CONFIG={"projectId":"foo", "locationId":"us-central1"}`
)

func TestAcceptance(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without package",
			App:  "no_package",
		},
		{
			Name: "function without package and with yarn",
			App:  "no_package_yarn",
		},
		{
			Name:       "function without framework",
			App:        "no_framework",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function without framework and with yarn",
			App:        "no_framework_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "function with framework",
			App:        "with_framework",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with dependencies",
			App:        "with_dependencies",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with dependencies and with yarn",
			App:        "with_dependencies_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "function with local dependency",
			App:        "local_dependency",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with local dependency and with yarn",
			App:        "local_dependency_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "function with gcp-build",
			App:        "with_gcp_build",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with runtime env var",
			App:        "with_env_var",
			RunEnv:     []string{"FOO=foo"},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with prepare",
			App:        "with_prepare",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "framework loads user dependencies",
			App:        "load_user_dependencies",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/execute"

			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_SIGNATURE_TYPE=http",
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=nodejs8",
				"X_GOOGLE_TARGET_PLATFORM=gcf",
			)

			// The legacy worker.js runtimes have a slightly different runtime
			// contract than the Function Frameworks.
			tc.RunEnv = append(tc.RunEnv,
				// X_GOOGLE_LOAD_ON_START loads the user code immediately, rather
				// than waiting for the supervisor to call the /load endpoint.
				"X_GOOGLE_LOAD_ON_START=true",
				// By default the worker uses port 8091, but our acceptance tests
				// connect to port 8080
				"X_GOOGLE_WORKER_PORT=8080",
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
			MustMatch: "SyntaxError:",
		},
		{
			App:       "fail_wrong_main",
			MustMatch: "function.js does not exist",
		},
		{
			App:       "with_framework_yarn",
			MustMatch: `The engine "node" is incompatible with this module`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=nodejs8",
				"X_GOOGLE_TARGET_PLATFORM=gcf",
			)

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
