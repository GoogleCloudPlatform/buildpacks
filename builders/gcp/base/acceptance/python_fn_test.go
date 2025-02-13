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

func TestAcceptancePythonFn(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "function without framework",
			App:        "without_framework",
			Path:       "/testFunction",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustUse:    []string{pythonRuntime, pythonFF, pythonPIP},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with custom source file",
			App:        "custom_file",
			Path:       "/testFunction",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_FUNCTION_SOURCE=func.py", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustUse:    []string{pythonRuntime, pythonFF, pythonPIP},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with dependencies",
			App:        "with_dependencies",
			Path:       "/testFunction",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustUse:    []string{pythonRuntime, pythonPIP, pythonFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with framework",
			App:        "with_framework",
			Path:       "/testFunction",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustUse:    []string{pythonRuntime, pythonPIP, pythonFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with runtime env var",
			App:        "with_env_var",
			Path:       "/testFunction",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_PYTHON_VERSION=3.11.0"},
			RunEnv:     []string{"FOO=foo"},
			MustUse:    []string{pythonRuntime, pythonFF, pythonPIP},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresPythonFn(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "missing framework file",
			App:       "with_framework",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_FUNCTION_SOURCE=func.py", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustMatch: `GOOGLE_FUNCTION_SOURCE specified file "func.py" but it does not exist`,
		},
		{
			Name:      "missing main.py",
			App:       "custom_file",
			Env:       []string{"GOOGLE_FUNCTION_TARGET=testFunction", "GOOGLE_PYTHON_VERSION=3.11.0"},
			MustMatch: "missing main.py and GOOGLE_FUNCTION_SOURCE not specified. Either create the function in main.py or specify GOOGLE_FUNCTION_SOURCE to point to the file that contains the function",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
