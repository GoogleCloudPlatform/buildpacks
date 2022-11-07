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

const ffJarPath = "/layers/google.java.functions-framework/functions-framework/functions-framework.jar"

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "function with maven",
			App:             "maven",
			Env:             []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			FilesMustExist:  []string{ffJarPath},
			EnableCacheTest: true,
		},
		{
			Name:           "function with build.finalName setting in pom.xml",
			App:            "maven_custom_name",
			Env:            []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			FilesMustExist: []string{ffJarPath},
		},
		{
			Name:              "function with invoker as maven dependency",
			App:               "maven_invoker_dep",
			Env:               []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			FilesMustExist:    []string{"/workspace/target/_javaInvokerDependency/java-function-invoker-1.0.2.jar"},
			FilesMustNotExist: []string{ffJarPath},
		},
		{
			Name:            "function with gradle",
			App:             "gradle",
			Env:             []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			FilesMustExist:  []string{ffJarPath},
			EnableCacheTest: true,
		},
		{
			Name:              "function with invoker as gradle dependency",
			App:               "gradle_invoker_dep",
			Env:               []string{"GOOGLE_FUNCTION_TARGET=functions.HelloWorld"},
			FilesMustExist:    []string{"/workspace/build/_javaFunctionDependencies/java-function-invoker-1.0.2.jar"},
			FilesMustNotExist: []string{ffJarPath},
		},
		{
			Name: "prebuilt jar",
			App:  "jar",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=functions.jar.HelloWorld"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		tc.FlakyBuildAttempts = 3

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/"
			tc.Env = append(tc.Env,
				"GOOGLE_RUNTIME=java11",
			)
			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")

			tc.FilesMustExist = append(tc.FilesMustExist,
				"/layers/google.utils.archive-source/src/source-code.tar.gz",
				"/workspace/.googlebuild/source-code.tar.gz",
			)

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
			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gcf")

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
