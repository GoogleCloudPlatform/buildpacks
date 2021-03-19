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

func TestAcceptanceCppFn(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "function without a namespace",
			App:        "no_namespace",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test_function"},
			Path:       "/test_function",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function in a namespace",
			App:        "with_namespace",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test::ns0::ns1::test_function"},
			Path:       "/test_function",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with its own CMake file",
			App:        "with_cmakelist",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test_function"},
			Path:       "/test_function",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with its own CMake file and vcpkg manifest",
			App:        "with_manifest",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test_function"},
			Path:       "/test_function",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		// TODO(b/183222659): Re-enable when fixed.
		tc.SkipCacheTest = true
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
