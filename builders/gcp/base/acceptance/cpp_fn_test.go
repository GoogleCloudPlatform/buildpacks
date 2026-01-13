// Copyright 2021 Google LLC
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

func TestAcceptanceCppFn(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			// Cpp is not supported on new stacks starting Ubuntu 22.04.
			SkipStacks: []string{"google.22", "google.min.22", "google.gae.22"},
			Name:       "function with additional dependencies",
			App:        "test_function",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test_function", "GOOGLE_FUNCTION_SIGNATURE_TYPE=http"},
			Path:       "/test_function",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
		{
			// Cpp is not supported on new stacks starting Ubuntu 22.04.
			SkipStacks: []string{"google.22", "google.min.22", "google.gae.22"},
			Name:       "function using declarative configuration",
			App:        "test_declarative",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=test_function"},
			Path:       "/test_declarative",
			MustUse:    []string{cppFF},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
