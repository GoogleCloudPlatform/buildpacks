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
//   apphosting/runtime/titanium/buildpacks/tools/run-acceptance-tests.sh --runtime=gcpbase
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
			Name:              "simple Go application",
			App:               "go/simple",
			MustUse:           []string{goRuntime, goBuild},
			FilesMustExist:    []string{"/layers/google.go.build/bin/main"},
			FilesMustNotExist: []string{"/layers/google.go.runtime"},
		},
		{
			Name:           "simple Go application (Dev Mode)",
			App:            "go/simple",
			Env:            []string{"GOOGLE_DEVMODE=1"},
			MustUse:        []string{goRuntime, goBuild},
			FilesMustExist: []string{"/layers/google.go.runtime/go/bin/go"},
		},
		{
			Name:    "Go runtime version respected",
			App:     "go/simple",
			Path:    "/version?want=1.13",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=1.13"},
			MustUse: []string{goRuntime, goBuild},
		},
		{
			Name:       "Go selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=go"},
			MustUse:    []string{goRuntime},
			MustNotUse: []string{nodeRuntime, pythonRuntime},
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
			Name:      "bad runtime version",
			App:       "go/simple",
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
