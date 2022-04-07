// Copyright 2022 Google LLC
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
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

const (
	goBuild       = "google.go.build"
	goClearSource = "google.go.clear_source"
	goFF          = "google.go.functions-framework"
	goMod         = "google.go.gomod"
	goPath        = "google.go.gopath"
	goRuntime     = "google.go.runtime"
)

func init() {
	acceptance.DefineFlags()
}

// TestGCPAcceptanceGo runs each GCP acceptance test case against each version of go.
func TestGCPAcceptanceGo(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []struct {
		Test               acceptance.Test
		ExcludedGoVersions []string
	}{
		{
			Test: acceptance.Test{
				Name:           "simple Go application",
				App:            "go/simple",
				MustUse:        []string{goRuntime, goBuild, goPath},
				MustNotUse:     []string{goClearSource},
				FilesMustExist: []string{"/layers/google.go.build/bin/main", "/workspace/main.go"},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "Go.mod",
				App:        "go/simple_gomod",
				MustUse:    []string{goRuntime, goBuild, goMod},
				MustNotUse: []string{goPath},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "Go.mod package",
				App:        "go/gomod_package",
				MustUse:    []string{goRuntime, goBuild, goMod},
				MustNotUse: []string{goPath},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "Multiple entrypoints",
				App:        "go/entrypoints",
				Env:        []string{"GOOGLE_BUILDABLE=cmd/first/main.go"},
				MustUse:    []string{goRuntime, goBuild},
				MustNotUse: []string{goPath},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "Go.mod and vendor",
				App:        "go/simple_gomod_vendor",
				MustUse:    []string{goRuntime, goBuild, goMod},
				MustNotUse: []string{goPath},
			},
			// go mod and vendor cannot be used together before go 1.14
			ExcludedGoVersions: []string{"1.11", "1.12", "1.13"},
		},
	}

	for _, tc := range testCases {
		for _, v := range goVersions {
			if shouldSkipVersion(v, tc.ExcludedGoVersions) {
				continue
			}
			verTC := applyRuntimeVersion(t, tc.Test, v)
			t.Run(verTC.Name, func(t *testing.T) {
				t.Parallel()

				acceptance.TestApp(t, builder, verTC)
			})
		}
	}
}

func shouldSkipVersion(version string, excludedVersions []string) bool {
	for _, ev := range excludedVersions {
		if version == ev {
			return true
		}
	}
	return false
}

// TestGCPAcceptanceGoSingleVersion runs GCP test cases which do not need to be tested against more
// than one go version.
func TestGCPAcceptanceGoSingleVersion(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:                "Dev mode",
			App:                 "go/simple",
			Env:                 []string{"GOOGLE_DEVMODE=1"},
			MustUse:             []string{goRuntime, goBuild, goPath},
			FilesMustExist:      []string{"/layers/google.go.runtime/go/bin/go", "/workspace/main.go"},
			MustRebuildOnChange: "/workspace/main.go",
		},
		{
			// This is a separate test case from Dev mode above because it has a fixed runtime version.
			// Its only purpose is to test that the metadata is set correctly.
			Name:    "Dev mode metadata",
			App:     "go/simple",
			Env:     []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=1.16.4"},
			MustUse: []string{goRuntime, goBuild, goPath},
			BOM: []acceptance.BOMEntry{
				{
					Name: "go",
					Metadata: map[string]interface{}{
						"version": "1.16.4",
					},
				},
				{
					Name: "devmode",
					Metadata: map[string]interface{}{
						"devmode.sync": []interface{}{
							map[string]interface{}{"dest": "/workspace", "src": "**/*.go"},
						},
					},
				},
			},
		},
		{
			Name:    "Go runtime version respected",
			App:     "go/simple",
			Path:    "/version?want=1.13",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=1.13"},
			MustUse: []string{goRuntime, goBuild, goPath},
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

func TestGCPFailuresGo(t *testing.T) {
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
			Name:                   "no Go files in root (Go 1.12)",
			App:                    "go/entrypoints",
			Env:                    []string{"GOOGLE_RUNTIME_VERSION=1.12"},
			MustMatch:              `Tip: "GOOGLE_BUILDABLE" env var configures which Go package is built`,
			SkipBuilderOutputMatch: true,
		},
		{
			Name:                   "no Go files in root (Go 1.14)",
			App:                    "go/entrypoints",
			Env:                    []string{"GOOGLE_RUNTIME_VERSION=1.14"},
			MustMatch:              `Tip: "GOOGLE_BUILDABLE" env var configures which Go package is built`,
			SkipBuilderOutputMatch: true,
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
