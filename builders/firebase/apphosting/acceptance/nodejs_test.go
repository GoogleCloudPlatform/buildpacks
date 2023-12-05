// Copyright 2023 Google LLC
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

var baseEnv = "FIREBASE_OUTPUT_BUNDLE_DIR=.apphosting/"

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceNodeJs(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "simple application",
			App:             "simple",
			MustUse:         []string{nodeRuntime, nodeNPM},
			EnableCacheTest: true,
		},
		{
			// Tests a specific versions of Node.js available on dl.google.com.
			Name:    "runtime version 16.17.1",
			App:     "simple",
			Path:    "/version?want=16.17.1",
			Env:     []string{"GOOGLE_NODEJS_VERSION=16.17.1"},
			MustUse: []string{nodeRuntime},
		},

		// TODO(b/315008858) This should be reenabled once yarn support is re added
		/*
			{
				Name:       "yarn",
				App:        "yarn",
				MustUse:    []string{nodeRuntime, nodeYarn},
				MustNotUse: []string{nodeNPM, nodePNPM},
			},
		*/
		{
			Name:       "pnpm",
			App:        "pnpm",
			MustUse:    []string{nodeRuntime, nodePNPM},
			MustNotUse: []string{nodeNPM, nodeYarn},
		},
		{
			Name:       "runtime version with npm ci",
			App:        "simple",
			Path:       "/version?want=16.18.1",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=16.18.1"},
			MustUse:    []string{nodeRuntime, nodeNPM},
			MustNotUse: []string{nodePNPM, nodeYarn},
		},
		{
			Name: "NPM version specified",
			// npm@8 requires nodejs@12+
			VersionInclusionConstraint: ">= 12.0.0",
			App:                        "npm_version_specified",
			MustOutput:                 []string{"npm --version\n\n8.3.1"},
			Path:                       "/version?want=8.3.1",
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc.Env = append(tc.Env, baseEnv)
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresNodeJs(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "invalid Node.js version specified",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
