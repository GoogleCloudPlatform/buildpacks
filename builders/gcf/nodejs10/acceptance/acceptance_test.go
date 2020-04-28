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
	npm          = "google.nodejs.npm"
	yarn         = "google.nodejs.yarn"
	npmGCPBuild  = "google.nodejs.npm-gcp-build"
	yarnGCPBuild = "google.nodejs.yarn-gcp-build"

	// Firebase functions expect FIREBASE_CONFIG & GCLOUD_PROJECT env vars at run time.
	// Otherwise, initializing the Firebase Admin SDK would fail.
	gcloudProject  = "GCLOUD_PROJECT=foo"
	firebaseConfig = `FIREBASE_CONFIG={"projectId":"foo", "locationId":"us-central1"}`
)

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
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
			Name:       "function with framework and with yarn",
			App:        "with_framework_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
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
			MustUse:    []string{npmGCPBuild},
			MustNotUse: []string{yarnGCPBuild},
		},
		{
			Name:       "function with gcp-build and with yarn",
			App:        "with_gcp_build_yarn",
			MustUse:    []string{yarnGCPBuild},
			MustNotUse: []string{npmGCPBuild},
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
			Name:       "function with prepare and with yarn",
			App:        "with_prepare_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "framework loads user dependencies",
			App:        "load_user_dependencies",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "firebase function pre v3.4.0",
			App:        "firebase_pre_3.4.0",
			RunEnv:     []string{gcloudProject, firebaseConfig},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "firebase function pre v3.4.0 with yarn",
			App:        "firebase_pre_3.4.0_yarn",
			RunEnv:     []string{gcloudProject, firebaseConfig},
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "firebase function post v3.4.0",
			App:        "firebase_post_3.4.0",
			RunEnv:     []string{gcloudProject, firebaseConfig},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "firebase function post v3.4.0 with yarn",
			App:        "firebase_post_3.4.0_yarn",
			RunEnv:     []string{gcloudProject, firebaseConfig},
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=nodejs10",
			)

			acceptance.TestApp(t, builder, tc)
		})
	}
}

// TODO: For compatibility with GCF; this will be removed later.
func TestAcceptanceOldEnv(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without package",
			App:  "no_package",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"FUNCTION_TARGET=testFunction",
				"GOOGLE_RUNTIME=nodejs10",
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
			Env:       []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=nodejs10"},
			MustMatch: "SyntaxError:",
		},
		{
			App:       "fail_wrong_main",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_RUNTIME=nodejs10"},
			MustMatch: "function.js does not exist",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
