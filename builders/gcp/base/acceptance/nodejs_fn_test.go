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

// Run these tests using:
// 	blaze test //third_party/gcp_buildpacks/builders/gcp/base/acceptance:nodejs_fn_test
//
// Note, you may need to update your stack images:
// 	third_party/gcp_buildpacks/tools/pull-images.sh gcp base
//
package acceptance

import (
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
			Name:       "function without package",
			App:        "no_package",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeFF},
			MustNotUse: []string{nodeNPM, nodeYarn, entrypoint},
		},
		{
			Name:       "function without package and with yarn",
			App:        "no_package_yarn",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeFF},
			MustNotUse: []string{nodeNPM, entrypoint},
		},
		{
			Name:       "function without framework",
			App:        "no_framework",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeNPM, nodeFF},
			MustNotUse: []string{nodeYarn, entrypoint},
		},
		{
			Name:       "function without framework and with yarn",
			App:        "no_framework_yarn",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeYarn, nodeFF},
			MustNotUse: []string{nodeNPM, entrypoint},
		},
		{
			Name:       "function with framework",
			App:        "with_framework",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeNPM, nodeFF},
			MustNotUse: []string{nodeYarn, entrypoint},
		},
		{
			Name:       "function with framework and with yarn",
			App:        "with_framework_yarn",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeYarn, nodeFF},
			MustNotUse: []string{nodeNPM, entrypoint},
		},
		{
			Name:       "function with dependencies",
			App:        "with_dependencies",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeNPM, nodeFF},
			MustNotUse: []string{nodeYarn, entrypoint},
		},
		{
			Name:       "function with dependencies and with yarn",
			App:        "with_dependencies_yarn",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			MustUse:    []string{nodeRuntime, nodeYarn, nodeFF},
			MustNotUse: []string{nodeNPM, entrypoint},
		},
		{
			Name:       "function with runtime env var",
			App:        "with_env_var",
			Path:       "/testFunction",
			Env:        []string{"FUNCTION_TARGET=testFunction"},
			RunEnv:     []string{"FOO=foo"},
			MustUse:    []string{nodeNPM},
			MustNotUse: []string{nodeYarn},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

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
			Env:       []string{"FUNCTION_TARGET=testFunction"},
			MustMatch: "SyntaxError:",
		},
		{
			App:       "fail_wrong_main",
			Env:       []string{"FUNCTION_TARGET=testFunction"},
			MustMatch: "Error: Cannot find module",
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
