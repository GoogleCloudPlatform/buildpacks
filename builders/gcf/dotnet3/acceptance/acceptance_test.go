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

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	defer cleanup()

	testCases := []acceptance.Test{
		{
			Name: "cs single target",
			App:  "cs_single_target",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=TestFunction.Function"},
			Path: "/function",
		},
		{
			Name: "cs multiple targets",
			App:  "cs_multiple_targets",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=TestFunction.Function"},
			Path: "/function",
		},
		{
			Name: "fs function",
			App:  "fs_function",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=fs_function.Function"},
			Path: "/function",
		},
		{
			Name: "vb function",
			App:  "vb_function",
			Env:  []string{"GOOGLE_FUNCTION_TARGET=vb_function.CloudFunction"},
			Path: "/function",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Env = append(tc.Env,
				"GOOGLE_RUNTIME=dotnet3",
			)

			acceptance.TestApp(t, builder, tc)
		})
	}
}
