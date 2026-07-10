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

var baseEnv = []string{
	"FIREBASE_OUTPUT_BUNDLE_DIR=.apphosting/",
	"X_GOOGLE_TARGET_PLATFORM=fah",
}

const (
	// Buildpack identifiers used to verify that buildpacks were or were not used.
	nodeNPM               = "google.nodejs.npm"
	nodePNPM              = "google.nodejs.pnpm"
	nodeRuntime           = "google.nodejs.runtime"
	nodeYarn              = "google.nodejs.yarn"
	nodeFirebaseNextJs    = "google.nodejs.firebasenextjs"
	nodeFirebaseAngular   = "google.nodejs.firebaseangular"
	nodeFirebaseNx        = "google.nodejs.firebasenx"
	nodeTurborepo         = "google.nodejs.turborepo"
	nodeFirebaseBundle    = "google.nodejs.firebasebundle"
	genericFirebaseBundle = "google.firebase.firebasebundle"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceNodeJs(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "simple application",
			App:             "npm",
			MustUse:         []string{nodeRuntime, nodeNPM, nodeFirebaseBundle},
			EnableCacheTest: true,
		},
		{
			// Tests a specific versions of Node.js available on dl.google.com.
			Name:    "runtime version 20.19.5",
			App:     "npm",
			Path:    "/version?want=20.19.5",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=20.19.5"},
			MustUse: []string{nodeRuntime, nodeFirebaseBundle},
			// Make sure the test run only against the google-22-full builder stack.
			SkipStacks: []string{"google.24.full", "google.24"},
			// Restrict the test to run only against the nodejs20 runtime version.
			VersionInclusionConstraint: "20",
		},
		{
			Name:       "yarn",
			App:        "yarn",
			MustUse:    []string{nodeRuntime, nodeYarn, nodeFirebaseBundle},
			MustNotUse: []string{nodeNPM, nodePNPM},
		},
		{
			Name:       "pnpm",
			App:        "pnpm",
			MustUse:    []string{nodeRuntime, nodePNPM, nodeFirebaseBundle},
			MustNotUse: []string{nodeNPM, nodeYarn},
		},
		{
			Name:       "runtime version with npm ci",
			App:        "npm",
			Path:       "/version?want=20.19.5",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=20.19.5"},
			MustUse:    []string{nodeRuntime, nodeNPM, nodeFirebaseBundle},
			MustNotUse: []string{nodePNPM, nodeYarn},
			// Make sure the test run only against the google-22-full builder stack.
			SkipStacks: []string{"google.24.full", "google.24"},
			// Restrict the test to run only against the nodejs20 runtime version.
			VersionInclusionConstraint: "20",
		},
		{
			Name: "NPM version specified",
			// npm@8 requires nodejs@12+
			VersionInclusionConstraint: ">= 12.0.0",
			App:                        "npm_version_specified",
			MustOutput:                 []string{"npm --version\n\n8.3.1"},
			Path:                       "/version?want=8.3.1",
		},
		{
			Name: "nextjs with npm",
			App:  "nextjs_npm",
			Env:  []string{"GOOGLE_BUILDABLE=./"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodeNPM,
				nodeFirebaseBundle,
			},
			MustNotUse: []string{nodeYarn, nodePNPM, nodeFirebaseAngular, nodeFirebaseNx, nodeTurborepo},
			MustOutput: []string{
				// Confirms Next.js build process completed successfully, and the firebasebundle buildpack
				// sets the run command correctly.
				"✓ Compiled successfully",
				"Setting run command from bundle.yaml: node .next/standalone/server.js",
			},
			MustMatchStatusCode: 200,
		},
		{
			Name: "nextjs with pnpm",
			App:  "nextjs_pnpm",
			Env:  []string{"GOOGLE_BUILDABLE=./"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodePNPM,
				nodeFirebaseBundle,
			},
			MustNotUse: []string{nodeYarn, nodeNPM, nodeFirebaseAngular, nodeFirebaseNx, nodeTurborepo},
			MustOutput: []string{
				"✓ Compiled successfully",
				"Setting run command from bundle.yaml: node .next/standalone/server.js",
			},
			MustMatchStatusCode: 200,
		},
		{
			Name: "nextjs with yarn",
			App:  "nextjs_yarn",
			Env:  []string{"GOOGLE_BUILDABLE=./"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodeYarn,
				nodeFirebaseBundle,
			},
			MustNotUse: []string{nodeNPM, nodePNPM, nodeFirebaseAngular, nodeFirebaseNx, nodeTurborepo},
			MustOutput: []string{
				"✓ Compiled successfully",
				"Setting run command from bundle.yaml: node .next/standalone/server.js",
			},
			MustMatchStatusCode: 200,
		},
		{
			Name: "nextjs turborepo with npm",
			App:  "nextjs_turbo",
			Env:  []string{"GOOGLE_BUILDABLE=apps/web"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodeTurborepo,
				nodeNPM,
				nodeFirebaseBundle,
			},
			MustNotUse: []string{nodePNPM, nodeYarn, nodeFirebaseAngular, nodeFirebaseNx},
			// confirms the test app runs successfully and serves the expected response
			MustMatch: "Hello from Next.js Turborepo on FAH!",
			MustOutput: []string{
				// Confirms Turborepo ran the 'next build' command for the 'web' package, and the
				// firebasebundle buildpack sets the run command correctly.
				"web:build: > next build",
				"Setting run command from bundle.yaml: node apps/web/.next/standalone/apps/web/server.js",
			},
			MustMatchStatusCode: 200,
		},
		{
			Name: "nextjs with npm and generic bundle enabled",
			App:  "nextjs_npm",
			Env:  []string{"GOOGLE_BUILDABLE=./", "GOOGLE_USE_GENERIC_FIREBASEBUNDLE=true"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodeNPM,
				genericFirebaseBundle,
			},
			MustNotUse: []string{nodeYarn, nodePNPM, nodeFirebaseAngular, nodeFirebaseNx, nodeTurborepo, nodeFirebaseBundle},
			MustOutput: []string{
				"✓ Compiled successfully",
				"Setting run command from bundle.yaml: node .next/standalone/server.js",
			},
			MustMatchStatusCode: 200,
		},
		{
			Name: "nextjs with npm and generic bundle disabled",
			App:  "nextjs_npm",
			Env:  []string{"GOOGLE_BUILDABLE=./", "GOOGLE_USE_GENERIC_FIREBASEBUNDLE=false"},
			MustUse: []string{
				nodeRuntime,
				nodeFirebaseNextJs,
				nodeNPM,
				nodeFirebaseBundle,
			},
			MustNotUse: []string{nodeYarn, nodePNPM, nodeFirebaseAngular, nodeFirebaseNx, nodeTurborepo, genericFirebaseBundle},
			MustOutput: []string{
				"✓ Compiled successfully",
				"Setting run command from bundle.yaml: node .next/standalone/server.js",
			},
			MustMatchStatusCode: 200,
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		// Prepend baseEnv to tc.Env, so that the baseEnv values can be overridden by the test case.
		tc.Env = append(baseEnv, tc.Env...)
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

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
			App:       "npm",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "invalid Node.js version specified",
		},
	}

	for _, tc := range testCases {
		tc.Env = append(baseEnv, tc.Env...)
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
