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
	entrypoint  = "google.config.entrypoint"
	nodeFF      = "google.nodejs.functions-framework"
	nodeNPM     = "google.nodejs.npm"
	nodeRuntime = "google.nodejs.runtime"
	nodeYarn    = "google.nodejs.yarn"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []struct {
		Test                   acceptance.Test
		IncludedNodeJSVersions []string
		ExcludedNodeJSVersions []string
	}{
		{
			Test: acceptance.Test{
				Name:    "simple application",
				App:     "nodejs/simple",
				MustUse: []string{nodeRuntime, nodeNPM},
			},
		},
		{
			Test: acceptance.Test{
				Name:                "Dev mode",
				App:                 "nodejs/simple",
				Env:                 []string{"GOOGLE_DEVMODE=1"},
				MustUse:             []string{nodeRuntime, nodeNPM},
				FilesMustExist:      []string{"/workspace/server.js"},
				MustRebuildOnChange: "/workspace/server.js",
			},
		},
		{
			// Test installs a specific version of node and only needs to be run with a single version
			IncludedNodeJSVersions: []string{"14.18.3"},
			Test: acceptance.Test{
				// This is a separate test case from Dev mode above because it has a fixed runtime version.
				// Its only purpose is to test that the metadata is set correctly.
				Name:    "Dev mode metadata",
				App:     "nodejs/simple",
				Env:     []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=14.17.0"},
				MustUse: []string{nodeRuntime, nodeNPM},
				BOM: []acceptance.BOMEntry{
					{
						Name: "node",
						Metadata: map[string]interface{}{
							"version": "14.17.0",
						},
					},
					{
						Name: "devmode",
						Metadata: map[string]interface{}{
							"devmode.sync": []interface{}{
								map[string]interface{}{"dest": "/workspace", "src": "**/*.js"},
								map[string]interface{}{"dest": "/workspace", "src": "**/*.mjs"},
								map[string]interface{}{"dest": "/workspace", "src": "**/*.coffee"},
								map[string]interface{}{"dest": "/workspace", "src": "**/*.litcoffee"},
								map[string]interface{}{"dest": "/workspace", "src": "**/*.json"},
								map[string]interface{}{"dest": "/workspace", "src": "public/**"},
							},
						},
					},
				},
			},
		},
		{
			Test: acceptance.Test{
				Name:    "simple application (custom entrypoint)",
				App:     "nodejs/custom_entrypoint",
				Env:     []string{"GOOGLE_ENTRYPOINT=node custom.js"},
				MustUse: []string{nodeRuntime, nodeNPM, entrypoint},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "yarn",
				App:        "nodejs/yarn",
				MustUse:    []string{nodeRuntime, nodeYarn},
				MustNotUse: []string{nodeNPM},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "yarn (Dev Mode)",
				App:        "nodejs/yarn",
				Env:        []string{"GOOGLE_DEVMODE=1"},
				MustUse:    []string{nodeRuntime, nodeYarn},
				MustNotUse: []string{nodeNPM},
			},
		},
		{
			// Test installs a specific version of node and only needs to be run with a single version
			IncludedNodeJSVersions: []string{"12.22.9"},
			Test: acceptance.Test{
				Name:    "runtime version with npm install",
				App:     "nodejs/simple",
				Path:    "/version?want=12.13.0",
				Env:     []string{"GOOGLE_RUNTIME_VERSION=12.13.0"},
				MustUse: []string{nodeRuntime, nodeNPM},
			},
		},
		{
			// Test installs a specific version of node and only needs to be run with a single version
			IncludedNodeJSVersions: []string{"12.22.9"},
			Test: acceptance.Test{
				Name:    "runtime version with npm ci",
				App:     "nodejs/simple",
				Path:    "/version?want=12.13.1",
				Env:     []string{"GOOGLE_RUNTIME_VERSION=12.13.1"},
				MustUse: []string{nodeRuntime, nodeNPM},
			},
		},
		{
			Test: acceptance.Test{
				Name:       "without package.json",
				App:        "nodejs/no_package",
				Env:        []string{"GOOGLE_ENTRYPOINT=node server.js"},
				MustUse:    []string{nodeRuntime},
				MustNotUse: []string{nodeNPM, nodeYarn},
			},
		},
		{
			Test: acceptance.Test{
				Name:    "selected via GOOGLE_RUNTIME",
				App:     "override",
				Env:     []string{"GOOGLE_RUNTIME=nodejs"},
				MustUse: []string{nodeRuntime},
			},
		},
		{
			// npm@7 is incompatible with nodejs@8
			ExcludedNodeJSVersions: []string{"8.17.0"},
			Test: acceptance.Test{
				Name:          "NPM version specified",
				App:           "nodejs/npm_version_specified",
				MustMatch:     "hello, world",
				MustOutput:    []string{"npm --version\n\n7.24.1"},
				SkipCacheTest: true,
			},
		},
	}

	for _, tc := range testCases {
		if !acceptance.ShouldTestVersion(tc.IncludedNodeJSVersions, tc.ExcludedNodeJSVersions) {
			continue
		}
		tc := tc
		t.Run(tc.Test.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc.Test)
		})
	}
}

func TestFailures(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []struct {
		FailureTest            acceptance.FailureTest
		IncludedNodeJSVersions []string
	}{
		{
			// Test has no version specific characteristics and should act the same across all versions
			IncludedNodeJSVersions: []string{"12.22.9"},
			FailureTest: acceptance.FailureTest{
				Name:      "bad runtime version",
				App:       "nodejs/simple",
				Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
				MustMatch: "Runtime version BAD_NEWS_BEARS does not exist",
			},
		},
	}

	for _, tc := range testCases {
		if !acceptance.ShouldTestVersion(tc.IncludedNodeJSVersions, nil) {
			continue
		}
		tc := tc
		t.Run(tc.FailureTest.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builder, tc.FailureTest)
		})
	}
}
