// Copyright 2023 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

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
		// Test that we can build a gomod app with no dependencies.
		// This does not include a main package provided by the stager.
		{
			Name:       "gomod no dependencies",
			App:        "gomod",
			MustOutput: []string{"go.sum not found, generating"},
			MustUse:    []string{flex, flexGoMod},
		},
		// Test that we can build a gomod app with a given go.sum.
		{
			Name:          "gomod go.sum",
			App:           "gomod_go_sum",
			MustNotOutput: []string{"go.sum not found, generating"},
			MustUse:       []string{flex, flexGoMod},
		},

		// Test that GOOGLE_BUILDABLE takes precedence over go-app-stager.
		{
			Name:       "gomod GOOGLE_BUILDABLE vs go-app-stager main package",
			App:        "gomod_wrong_stager_mainpackage",
			Env:        []string{"GOOGLE_BUILDABLE=./maindir"},
			MustUse:    []string{flex},
			MustNotUse: []string{flexGoMod},
		},
		// Test that a gomod app can build main package from go-app-stager.
		{
			Name:    "gomod stager main",
			App:     "gomod_stager_mainpackage",
			MustUse: []string{flex, flexGoMod},
		},
	}

	for _, tc := range testCases {
		tc := applyStaticTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	if tc.Name == "" {
		tc.Name = tc.App
	}
	tc.Env = append(tc.Env, []string{"X_GOOGLE_TARGET_PLATFORM=flex", "GAE_APPLICATION_YAML_PATH=app.yaml"}...)
	return tc
}
