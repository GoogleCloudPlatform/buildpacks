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

func TestAcceptanceNodeJs(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "simple application",
			App:     "nodejs/simple",
			MustUse: []string{nodeRuntime, nodeNPM},
		},
		{
			Name:    "simple application (Dev Mode)",
			App:     "nodejs/simple",
			Env:     []string{"GOOGLE_DEVMODE=1"},
			MustUse: []string{nodeRuntime, nodeNPM},
		},
		{
			Name:    "simple application (custom entrypoint)",
			App:     "nodejs/custom_entrypoint",
			Env:     []string{"GOOGLE_ENTRYPOINT=node custom.js"},
			MustUse: []string{nodeRuntime, nodeNPM, entrypoint},
		},
		{
			Name:       "yarn",
			App:        "nodejs/yarn",
			MustUse:    []string{nodeRuntime, nodeYarn},
			MustNotUse: []string{nodeNPM},
		},
		{
			Name:       "yarn (Dev Mode)",
			App:        "nodejs/yarn",
			Env:        []string{"GOOGLE_DEVMODE=1"},
			MustUse:    []string{nodeRuntime, nodeYarn},
			MustNotUse: []string{nodeNPM},
		},
		{
			Name:    "runtime version with npm install",
			App:     "nodejs/simple",
			Path:    "/version?want=12.13.0",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=12.13.0"},
			MustUse: []string{nodeRuntime, nodeNPM},
		},
		{
			Name:    "runtime version with npm ci",
			App:     "nodejs/simple",
			Path:    "/version?want=12.13.1",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=12.13.1"},
			MustUse: []string{nodeRuntime, nodeNPM},
		},
		{
			Name:       "without package.json",
			App:        "nodejs/no_package",
			Env:        []string{"GOOGLE_ENTRYPOINT=node server.js"},
			MustUse:    []string{nodeRuntime},
			MustNotUse: []string{nodeNPM, nodeYarn},
		},
		{
			Name:       "selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=nodejs"},
			MustUse:    []string{nodeRuntime},
			MustNotUse: []string{goRuntime, javaRuntime, pythonRuntime},
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

func TestFailuresNodeJs(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "nodejs/simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "Runtime version BAD_NEWS_BEARS does not exist",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
