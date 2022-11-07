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
	"flag"
	"os"
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
			Name: "function with maven",
			App:  "maven",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
		},
		{
			Name: "function with gradle",
			App:  "gradle",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
		},
		{
			Name:              "function with clear source maven",
			App:               "maven",
			Env:               []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld", "GOOGLE_CLEAR_SOURCE=true"},
			FilesMustNotExist: []string{"/workspace/src/main/java/functions/HelloWorld.java", "/workspace/pom.xml"},
		},
		{
			Name:              "function with clear source gradle",
			App:               "gradle",
			Env:               []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld", "GOOGLE_CLEAR_SOURCE=true"},
			FilesMustNotExist: []string{"/workspace/src/main/java/functions/HelloWorld.java", "/workspace/build.gradle"},
		},
		{
			Name: "prebuilt jar",
			App:  "jar",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.jar.HelloWorld"},
		},
		{
			Name: "function with maven wrapper",
			App:  "mvnw",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		tc.FlakyBuildAttempts = 3

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "fail_syntax_error",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustMatch: `\[ERROR\].*not a statement`,
		},
		{
			App:       "fail_two_jars",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustMatch: "function has no pom.xml and more than one jar file: fatjar1.jar, fatjar2.jar",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	cleanup := acceptance.UnarchiveTestData()
	// We can't use defer cleanup() here because os.Exit prevents deferred functions from running.
	status := m.Run()
	cleanup()
	os.Exit(status)
}
