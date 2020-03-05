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
//   apphosting/runtime/titanium/buildpacks/tools/run-acceptance-tests.sh --runtime=gcfnodejs10
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
		},
		{
			Name: "function without framework",
			App:  "no_framework",
			Path: "/Func",
		},
		{
			Name: "vendored function without framwork",
			App:  "no_framework_vendored",
			Path: "/Func",
		},
		{
			Name: "function with framework",
			App:  "with_framework",
			Path: "/Func",
		},
		{
			Name: "function at /*",
			App:  "no_deps",
			Path: "/",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env,
				"FUNCTION_TARGET=Func",
				"GOOGLE_RUNTIME=go113",
			)

			acceptance.TestApp(t, builder, tc)
		})
	}
}
