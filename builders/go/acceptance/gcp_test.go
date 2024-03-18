// Copyright 2022 Google LLC
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
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "simple Go application",
			App:             "simple",
			MustUse:         []string{goRuntime, goBuild, goPath},
			MustNotUse:      []string{goClearSource},
			FilesMustExist:  []string{"/layers/google.go.build/bin/main", "/workspace/main.go"},
			EnableCacheTest: true,
		},
		{
			VersionInclusionConstraint: ">= 1.17",
			Name:                       "simple go app, no gomod with vendored deps",
			App:                        "simple_no_gomod_vendor",
			MustUse:                    []string{goRuntime, goBuild, goPath},
			MustNotUse:                 []string{goClearSource},
			EnableCacheTest:            true,
		},
		{
			Name:       "Go.mod",
			App:        "simple_gomod",
			MustUse:    []string{goRuntime, goBuild, goMod},
			MustNotUse: []string{goPath},
		},
		{
			Name:       "Go.mod package",
			App:        "gomod_package",
			MustUse:    []string{goRuntime, goBuild, goMod},
			MustNotUse: []string{goPath},
		},
		{
			Name:       "Multiple entrypoints",
			App:        "entrypoints",
			Env:        []string{"GOOGLE_BUILDABLE=cmd/first/main.go"},
			MustUse:    []string{goRuntime, goBuild},
			MustNotUse: []string{goPath},
		},
		{
			Name: "Go.mod and vendor",
			// go mod and vendor cannot be used together before go 1.14
			VersionInclusionConstraint: ">= 1.14",
			App:                        "simple_gomod_vendor",
			MustUse:                    []string{goRuntime, goBuild, goMod},
			MustNotUse:                 []string{goPath},
			EnableCacheTest:            true,
		},
		{
			Name: "Dev mode",
			// This test only runs against a single version of Go as it is unlikely to break across versions.
			VersionInclusionConstraint: "1.16",
			App:                        "simple",
			Env:                        []string{"GOOGLE_DEVMODE=1"},
			MustUse:                    []string{goRuntime, goBuild, goPath},
			FilesMustExist:             []string{"/layers/google.go.runtime/go/bin/go", "/workspace/main.go"},
			MustRebuildOnChange:        "/workspace/main.go",
		},
		{
			// This is a separate test case from Dev mode above because it has a fixed runtime version.
			// Its only purpose is to test that the metadata is set correctly.
			Name: "Dev mode metadata",
			// This test only runs against a single version of Go as it is unlikely to break across versions.
			VersionInclusionConstraint: "1.16",
			App:                        "simple",
			Env:                        []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=1.16.4"},
			MustUse:                    []string{goRuntime, goBuild, goPath},
			BOM: []acceptance.BOMEntry{
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
			Name: "Go runtime version respected",
			// This test only runs against a single version of Go as it is unlikely to break across versions.
			VersionInclusionConstraint: "1.13",
			App:                        "simple",
			Path:                       "/version?want=1.13",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=1.13"},
			MustUse:                    []string{goRuntime, goBuild, goPath},
		},
		{
			Name: "clear source",
			// This test only runs against a single version of Go as it is unlikely to break across versions.
			VersionInclusionConstraint: "1.16",
			App:                        "simple",
			Env:                        []string{"GOOGLE_CLEAR_SOURCE=true"},
			MustUse:                    []string{goClearSource},
			FilesMustExist:             []string{"/layers/google.go.build/bin/main"},
			FilesMustNotExist:          []string{"/layers/google.go.runtime", "/workspace/main.go"},
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

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:                   "no Go files in root",
			App:                    "entrypoints",
			MustMatch:              `Tip: "GOOGLE_BUILDABLE" env var configures which Go package is built`,
			SkipBuilderOutputMatch: true,
		},
		{
			Name: "bad runtime version",
			// This test only runs against a single version of Go as it is unlikely to break across versions.
			VersionInclusionConstraint: "1.16",
			App:                        "simple",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch:                  "invalid Go version specified:",
		},
		{
			Name:                       "no gomod; no vendor application",
			VersionInclusionConstraint: ">= 1.22",
			App:                        "simple_no_gomod_no_vendor_with_deps",
			SkipBuilderOutputMatch:     true,
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
