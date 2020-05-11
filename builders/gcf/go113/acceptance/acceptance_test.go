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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	gomod = "google.go.gomod"
)

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without deps",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/Func",
		},
		{
			Name:    "function without framework",
			App:     "no_framework",
			Env:     []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustUse: []string{gomod},
			Path:    "/Func",
		},
		{
			Name:       "vendored function without framwork",
			App:        "no_framework_vendored",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustNotUse: []string{gomod},
			Path:       "/Func",
		},
		{
			Name:    "function with framework",
			App:     "with_framework",
			Env:     []string{"GOOGLE_FUNCTION_TARGET=Func"},
			MustUse: []string{gomod},
			Path:    "/Func",
		},
		{
			Name: "function at /*",
			App:  "no_deps",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
			Path: "/",
		},
		{
			Name: "function with subdirectories",
			App:  "with_subdir",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=Func"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go113")

			acceptance.TestApp(t, builder, tc)
		})
	}
}
