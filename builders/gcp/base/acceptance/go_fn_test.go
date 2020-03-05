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
// 	blaze test //third_party/gcp_buildpacks/builders/gcp/base/acceptance:go_fn_test
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
			Name: "function without deps",
			App:  "no_deps",
			Path: "/Func",
			// Use Go 1.13 because Go 1.14 requires a go.mod file.
			Env:        []string{"FUNCTION_TARGET=Func", "GOOGLE_RUNTIME_VERSION=1.13.8"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function without framework",
			App:        "no_framework",
			Path:       "/Func",
			Env:        []string{"FUNCTION_TARGET=Func"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name: "vendored function without framwork",
			App:  "no_framework_vendored",
			Path: "/Func",
			// Use Go 1.13 because Go 1.14 requires a go.mod file.
			Env:        []string{"FUNCTION_TARGET=Func", "GOOGLE_RUNTIME_VERSION=1.13.8"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with framework",
			App:        "with_framework",
			Path:       "/Func",
			Env:        []string{"FUNCTION_TARGET=Func"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
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
