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

func TestAcceptanceNodeJs(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "simple application",
			App:     "nodejs/simple",
			MustUse: []string{nodeRuntime, nodeNPM},
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
			Name:    "Dev mode metadata",
			App:     "nodejs/simple",
			Env:     []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=14.17.0"},
			MustUse: []string{nodeRuntime, nodeNPM},
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
		{
			Name:    "runtime version with npm ci",
			App:     "nodejs/simple",
			Path:    "/version?want=16.9.1",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=16.9.1"},
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
		{
			Name:          "NPM version specified",
			App:           "nodejs/npm_version_specified",
			MustOutput:    []string{"npm --version\n\n8.3.1"},
			Path:          "/version?want=8.3.1",
			SkipCacheTest: true,
		},
		{
			Name:          "old NPM version specified",
			App:           "nodejs/old_npm_version_specified",
			Path:          "/version?want=5.5.1",
			MustUse:       []string{nodeRuntime, nodeNPM},
			MustOutput:    []string{"npm --version\n\n5.5.1"},
			SkipCacheTest: true,
		},
	}
	// Tests for specific versions of Node.js available on dl.google.com.
	for _, v := range []string{"8.17.0"} {
		testCases = append(testCases, acceptance.Test{
			Name:    "runtime version " + v,
			App:     "nodejs/simple",
			Path:    "/version?want=" + v,
			Env:     []string{"GOOGLE_NODEJS_VERSION=" + v},
			MustUse: []string{nodeRuntime},
		})
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func TestFailuresNodeJs(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "nodejs/simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "invalid Node.js version specified",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
