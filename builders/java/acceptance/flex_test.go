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
		// Test that we can build a maven project.
		{
			// Test application has been updated to use Java 17.
			// Flex applications will not be able to build Java 11 applications. We can remove this
			// constraint once Java 11 is deprecated.
			VersionInclusionConstraint: ">11",
			Name:                       "maven project springboot",
			App:                        "helloworld_springboot",
			Env:                        []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustUse:                    []string{"google.config.flex"},
		},
		// Test that we can build a gradle project
		{
			// Test application has been updated to use Java 17.
			// Flex applications will not be able to build Java 11 applications. We can remove this
			// constraint once Java 11 is deprecated.
			VersionInclusionConstraint: ">11 <25.0.0",
			Name:                       "gradle project",
			App:                        "gradle_quarkus",
			Env:                        []string{"GAE_APPLICATION_YAML_PATH=app.yaml"},
			MustUse:                    []string{"google.config.flex"},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
