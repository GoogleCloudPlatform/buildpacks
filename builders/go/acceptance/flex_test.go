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
		{
			Name:       "gomod no dependencies",
			App:        "gomod",
			Env:        []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustOutput: []string{"go.sum not found, generating"},
			MustUse:    []string{flex},
		},
		// Test that we can build a gomod app with a given go.sum.
		{
			Name:          "gomod go.sum",
			App:           "gomod_go_sum",
			Env:           []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustNotOutput: []string{"go.sum not found, generating"},
			MustUse:       []string{flex},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
