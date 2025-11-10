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

const (
	entrypoint    = "google.config.entrypoint"
	pythonFF      = "google.python.functions-framework"
	pythonPIP     = "google.python.pip"
	pythonRuntime = "google.python.runtime"
	pythonPoetry  = "google.python.poetry"
	pythonUV      = "google.python.uv"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:                       "function_without_framework",
			App:                        "without_framework",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonFF},
			MustNotOutput:              []string{`WARNING: You are using pip version`},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "function_without_framework_default_uv",
			App:                        "without_framework",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "function_without_framework_uv",
			App:                        "without_framework",
			MustNotOutput:              []string{`WARNING: You are using pip version`},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_without_framework_and_allow_injection",
			App:                        "without_framework",
			Env:                        []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=False"},
			MustNotOutput:              []string{`WARNING: You are using pip version`},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_without_framework_and_allow_injection_uv",
			App:                        "without_framework",
			Env:                        []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=False", "X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			MustNotOutput:              []string{`WARNING: You are using pip version`},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_with_dependencies",
			App:                        "with_dependencies",
			EnableCacheTest:            true,
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "function_with_dependencies_default_uv",
			App:                        "with_dependencies",
			EnableCacheTest:            true,
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "function_with_dependencies_uv",
			App:                        "with_dependencies",
			EnableCacheTest:            true,
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_with_framework",
			App:                        "with_framework",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "function_with_framework_default_uv",
			App:                        "with_framework",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "function_with_framework_uv",
			App:                        "with_framework",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_using_http_declarative_function_signatures",
			App:                        "use_declarative",
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "function_using_CloudEvent_declarative_function_signatures",
			App:                        "use_cloud_event_declarative",
			MustMatch:                  "OK",
			RequestType:                acceptance.CloudEventType,
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name: "function_with_framework_and_dependency_bin",
			App:  "with_framework_bin_conflict",
			// TODO(harisam): Remove this constraint once spacy support is added for python 3.13.
			VersionInclusionConstraint: ">= 3.8.0 < 3.13.0",
		},
		{
			Name:                       "function_with_runtime_env_var",
			App:                        "with_env_var",
			RunEnv:                     []string{"FOO=foo"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "function_with_runtime_env_var_default_uv",
			App:                        "with_env_var",
			RunEnv:                     []string{"FOO=foo"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "function_with_runtime_env_var_uv",
			App:                        "with_env_var",
			RunEnv:                     []string{"FOO=foo"},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name: "function_has_right_number_of_dependencies",
			App:  "list_dependencies",
			// The list_dependencies app has a dependency on the libexpat OS package which isn't installed
			// in the min run image.
			SkipStacks:                 []string{"google.min.22"},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "poetry",
			App:                        "poetry",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			VersionInclusionConstraint: ">=3.9.0",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonFF},
		},
		{
			Name:                       "pyproject",
			App:                        "pyproject",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			VersionInclusionConstraint: ">=3.9.0",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
		},
		{
			Name:                       "pyproject_pip",
			App:                        "pyproject",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			VersionInclusionConstraint: ">=3.9.0",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonFF},
		},
		{
			Name:                       "pyproject_uv",
			App:                        "pyproject",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			VersionInclusionConstraint: ">=3.9.0",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
		},
		{
			Name:                       "uv",
			App:                        "uv",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=BETA"},
			VersionInclusionConstraint: ">=3.9.0",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonFF},
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := applyStaticTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	tc.Path = "/testFunction"
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	tc.FilesMustExist = append(tc.FilesMustExist,
		"/layers/google.utils.archive-source/src/source-code.tar.gz",
		"/workspace/.googlebuild/source-code.tar.gz",
	)
	return tc
}

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "fail_syntax_error",
			MustMatch: "SyntaxError: invalid syntax",
		},
		{
			App:       "fail_broken_dependencies",
			MustMatch: `functions-framework .* has requirement flask.*,>=.*, but you have flask 0\.12\.5`,
			// this is only a warning in python37
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			App:                        "fail_broken_dependencies",
			MustMatch:                  "\".*The package \\`functions-framework\\` requires \\`flask>=.*,.*\\`.*, but \\`0.12.5\\` is installed.*\"",
			VersionInclusionConstraint: ">= 3.14.0", // For python 3.14+, the default package manager is uv.
		},
		{
			Name:      "function without dependencies or injection",
			App:       "without_framework",
			Env:       []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=True"},
			MustMatch: "skipping automatic framework injection has been enabled",
		},
		{
			Name:      "use pip vendored deps - framework not vendored",
			App:       "without_framework",
			Env:       []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=vendor"},
			MustMatch: "Vendored dependencies detected, please add functions-framework to requirements.txt and download it using pip",
		},
		{
			Name:      "fail_pyproject_without_framework",
			App:       "fail_pyproject_without_framework",
			Env:       []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustMatch: "This project is using pyproject.toml but you have not included the Functions Framework in your dependencies. Please add it to your pyproject.toml.",
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := applyStaticFailureTestOptions(tc)
		t.Run(tc.App, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}

func applyStaticFailureTestOptions(tc acceptance.FailureTest) acceptance.FailureTest {
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	return tc
}
