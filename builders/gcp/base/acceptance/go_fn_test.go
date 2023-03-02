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

func TestAcceptanceGoFn(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without deps",
			App:  "no_deps",
			Path: "/Func",
			// Use Go 1.13 because Go 1.14 requires a go.mod file.
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_RUNTIME_VERSION=1.13.8"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function without framework",
			App:        "no_framework_go_sum",
			Path:       "/Func",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name: "vendored function without framwork",
			App:  "no_framework_vendored_no_go_mod",
			Path: "/Func",
			// Use Go 1.13 because Go 1.14 requires a go.mod file.
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func", "GOOGLE_RUNTIME_VERSION=1.13.8"},
			MustUse:    []string{goRuntime, goFF, goBuild},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "function with framework",
			App:        "with_framework_go_sum",
			Path:       "/Func",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustUse:    []string{goRuntime, goFF, goBuild},
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
