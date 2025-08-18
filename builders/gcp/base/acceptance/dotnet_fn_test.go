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

func TestAcceptanceDotNetFn(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		// When there is only one target, we don't need to set FUNCTION_TARGET.
		{
			// .NET 3.1 is not supported on Ubuntu 22.04.
			Name:       "cs single target",
			App:        "cs_single_target",
			SkipStacks: []string{"google.24.full", "google.24", "google.22", "google.gae.22"},
			Path:       "/function",
			MustUse:    []string{dotnetRuntime, dotnetPublish},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "cs multiple targets",
			App:        "cs_multiple_targets",
			SkipStacks: []string{"google.24.full", "google.24"},
			Env:        []string{"GOOGLE_FUNCTION_TARGET=TestFunction.Function"},
			Path:       "/function",
			MustUse:    []string{dotnetRuntime, dotnetPublish, dotnetFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "fs function",
			App:        "fs_function",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=fs_function.Function"},
			Path:       "/function",
			MustUse:    []string{dotnetRuntime, dotnetPublish, dotnetFF},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "vb function",
			App:        "vb_function",
			Env:        []string{"GOOGLE_FUNCTION_TARGET=vb_function.CloudFunction"},
			Path:       "/function",
			MustUse:    []string{dotnetRuntime, dotnetPublish, dotnetFF},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
