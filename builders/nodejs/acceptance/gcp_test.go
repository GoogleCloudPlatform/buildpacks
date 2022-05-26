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
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "simple application",
			App:             "nodejs/simple",
			MustUse:         []string{nodeRuntime, nodeNPM},
			EnableCacheTest: true,
		},
		{
			Name:                "Dev mode",
			App:                 "nodejs/simple",
			Env:                 []string{"GOOGLE_DEVMODE=1"},
			MustUse:             []string{nodeRuntime, nodeNPM},
			FilesMustExist:      []string{"/workspace/server.js"},
			MustRebuildOnChange: "/workspace/server.js",
		},
		{
			// This is a separate test case from Dev mode above because it has a fixed runtime version.
			// Its only purpose is to test that the metadata is set correctly.
			Name: "Dev mode metadata",
			// Test installs a specific version of node and only needs to be run with a single version
			VersionInclusionConstraint: "14",
			App:                        "nodejs/simple",
			Env:                        []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=14.17.0"},
			MustUse:                    []string{nodeRuntime, nodeNPM},
			BOM: []acceptance.BOMEntry{
				{
					Name: "nodejs",
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
		// TODO (mattrobertson) update this to key off of the npm version
		// instead of the Node.js version.
		// {
		// 	// Test installs a specific version of node and only needs to be run with a single version
		// 	VersionInclusionConstraint: "12",
		// 	Test: acceptance.Test{
		// 		Name:    "runtime version with npm install",
		// 		App:     "nodejs/simple",
		// 		Path:    "/version?want=12.13.0",
		// 		Env:     []string{"GOOGLE_RUNTIME_VERSION=12.13.0"},
		// 		MustUse: []string{nodeRuntime, nodeNPM},
		// 	},
		// },
		{
			Name: "runtime version with npm ci",
			// Test installs a specific version of node and only needs to be run with a single version
			VersionInclusionConstraint: "12",
			App:                        "nodejs/simple",
			Path:                       "/version?want=16.9.1",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=16.9.1"},
			MustUse:                    []string{nodeRuntime, nodeNPM},
		},
		{
			Name:       "without package.json",
			App:        "nodejs/no_package",
			Env:        []string{"GOOGLE_ENTRYPOINT=node server.js"},
			MustUse:    []string{nodeRuntime},
			MustNotUse: []string{nodeNPM, nodeYarn},
		},
		{
			Name:    "selected via GOOGLE_RUNTIME",
			App:     "override",
			Env:     []string{"GOOGLE_RUNTIME=nodejs"},
			MustUse: []string{nodeRuntime},
		},
		{
			Name: "NPM version specified",
			// npm@8 requires nodejs@12+
			VersionInclusionConstraint: ">= 12.0.0",
			App:                        "nodejs/npm_version_specified",
			MustOutput:                 []string{"npm --version\n\n8.3.1"},
			Path:                       "/version?want=8.3.1",
		},
		{
			Name: "old NPM version specified",
			// npm@5 requires nodejs@8
			VersionInclusionConstraint: "8",
			App:                        "nodejs/old_npm_version_specified",
			Path:                       "/version?want=5.5.1",
			MustUse:                    []string{nodeRuntime, nodeNPM},
			MustOutput:                 []string{"npm --version\n\n5.5.1"},
		},
	}

	for _, tc := range acceptance.FilterTests(t, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name: "bad runtime version",
			// Test has no version specific characteristics and should act the same across all versions
			VersionInclusionConstraint: "12",
			App:                        "nodejs/simple",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch:                  "invalid Node.js version specified",
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
