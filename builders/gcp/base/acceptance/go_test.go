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

func TestAcceptanceGo(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:           "simple Go application",
			App:            "go/simple",
			MustUse:        []string{goRuntime, goBuild, goPath},
			MustNotUse:     []string{goClearSource},
			FilesMustExist: []string{"/layers/google.go.build/bin/main", "/workspace/main.go"},
		},
		{
			Name:       "Go.mod",
			App:        "go/simple_gomod",
			MustUse:    []string{goRuntime, goBuild},
			MustNotUse: []string{goPath},
		},
		{
			Name:       "Go.mod package",
			App:        "go/gomod_package",
			MustUse:    []string{goRuntime, goBuild},
			MustNotUse: []string{goPath},
		},
		{
			Name:       "Multiple entrypoints",
			App:        "go/entrypoints",
			Env:        []string{"GOOGLE_BUILDABLE=cmd/first/main.go"},
			MustUse:    []string{goRuntime, goBuild},
			MustNotUse: []string{goPath},
		},
		{
			Name:       "Go.mod and vendor",
			App:        "go/simple_gomod_vendor",
			MustUse:    []string{goRuntime, goBuild},
			MustNotUse: []string{goPath},
		},
		{
			Name:                "simple Go application (Dev Mode)",
			App:                 "go/simple",
			Env:                 []string{"GOOGLE_DEVMODE=1"},
			MustUse:             []string{goRuntime, goBuild, goPath},
			FilesMustExist:      []string{"/layers/google.go.runtime/go/bin/go", "/workspace/main.go"},
			MustRebuildOnChange: "/workspace/main.go",
		},
		{
			Name:    "Go runtime version respected",
			App:     "go/simple",
			Path:    "/version?want=1.13",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=1.13"},
			MustUse: []string{goRuntime, goBuild, goPath},
		},
		{
			Name:       "Go selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=go"},
			MustUse:    []string{goRuntime},
			MustNotUse: []string{javaRuntime, nodeRuntime, pythonRuntime},
		},
		{
			Name:              "clear source",
			App:               "go/simple",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true"},
			MustUse:           []string{goClearSource},
			FilesMustExist:    []string{"/layers/google.go.build/bin/main"},
			FilesMustNotExist: []string{"/layers/google.go.runtime", "/workspace/main.go"},
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

func TestFailuresGo(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "go/simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "Runtime version BAD_NEWS_BEARS does not exist",
		},
		{
			Name:      "no Go files in root (Go 1.12)",
			App:       "go/entrypoints",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=1.12"},
			MustMatch: `Tip: "GOOGLE_BUILDABLE" env var configures which Go package is built`,
		},
		{
			Name:      "no Go files in root (Go 1.14)",
			App:       "go/entrypoints",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=1.14"},
			MustMatch: `Tip: "GOOGLE_BUILDABLE" env var configures which Go package is built`,
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
