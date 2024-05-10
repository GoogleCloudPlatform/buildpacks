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
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}
func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)
	testCases := []acceptance.Test{
		// Test that we can build a ruby project.
		{
			Name:          "hello world project",
			App:           "helloworld",
			Env:           []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustNotOutput: []string{"WARNING"},
			MustUse:       []string{"google.config.flex", "google.ruby.flex-entrypoint"},
		},
		{
			Name:          "rails project",
			App:           "rails",
			Env:           []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustNotOutput: []string{"WARNING"},
			MustUse:       []string{"google.config.flex", "google.ruby.flex-entrypoint"},
		},
		{
			Name:          "rack project",
			App:           "rack",
			Env:           []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustNotOutput: []string{"WARNING"},
			MustUse:       []string{"google.config.flex", "google.ruby.flex-entrypoint"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=flex")
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
