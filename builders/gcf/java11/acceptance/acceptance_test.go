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
	"flag"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
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
			Name: "prebuilt jar",
			App:  "jar",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.jar.HelloWorld"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/"
			tc.Env = append(tc.Env,
				"GOOGLE_RUNTIME=java11",
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
			App:       "fail_syntax_error",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustMatch: `\[ERROR\].*not a statement`,
		},
		{
			App:       "fail_no_pom_no_jar",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustMatch: "function has neither pom.xml nor already-built jar file; directory has these entries: .googlebuild, random.txt",
		},
		{
			App:       "fail_two_jars",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			MustMatch: "function has no pom.xml and more than one jar file: fatjar1.jar, fatjar2.jar",
		},
		{
			App:       "maven",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.NonExistent"},
			MustMatch: `build succeeded but did not produce the class "functions.NonExistent"`,
		},
		{
			App:       "gradle",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.NonExistent"},
			MustMatch: `build succeeded but did not produce the class "functions.NonExistent"`,
		},
		{
			App:       "jar",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=functions.NonExistent"},
			MustMatch: `build succeeded but did not produce the class "functions.NonExistent"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env,
				"GOOGLE_RUNTIME=java11",
			)

			acceptance.TestBuildFailure(t, builder, tc)
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
