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

func init() {
	acceptance.DefineFlags()
}

const (
	npm  = "google.nodejs.npm"
	yarn = "google.nodejs.yarn"
	flex = "google.config.flex"
)

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	// New runtimes only support ubuntu22.
	testCases := []acceptance.Test{
		{
			App:        "npm_default",
			MustUse:    []string{npm, flex},
			MustNotUse: []string{yarn},
			SkipStacks: []string{"google.gae.18", "google.18"},
		},
		{
			App:        "yarn_specified",
			MustUse:    []string{yarn, flex},
			MustNotUse: []string{npm},
			SkipStacks: []string{"google.gae.18", "google.18"},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := applyStaticTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=flex")
	if tc.Name == "" {
		tc.Name = tc.App
	}
	return tc
}
