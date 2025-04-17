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

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "symfony app",
			App:  "symfony",
			// VersionInclusionConstraint: "< 8.2.0",
			MustUse:         []string{composer, composerInstall, phpRuntime},
			MustNotUse:      []string{composerGCPBuild, phpWebConfig, functionFramework, cloudFunctions},
			EnableCacheTest: true,
		},
		{
			Name: "composer.json without dependencies",
			App:  "composer_json_no_dependencies",
			// VersionInclusionConstraint: "< 8.2.0",
			MustUse:    []string{composer, composerInstall, phpRuntime},
			MustNotUse: []string{composerGCPBuild, phpWebConfig, functionFramework, cloudFunctions},
		},
		{
			Name: "composer.lock respected",
			App:  "composer_lock",
			// VersionInclusionConstraint: "< 8.2.0",
			MustUse:    []string{composer, composerInstall, phpRuntime},
			MustNotUse: []string{composerGCPBuild, phpWebConfig, functionFramework, cloudFunctions},
		},
		{
			Name: "composer.json with gcp-build script and no dependencies",
			App:  "gcp_build_no_dependencies",
			// VersionInclusionConstraint: "< 8.2.0",
			MustUse:    []string{composer, composerGCPBuild, composerInstall, phpRuntime},
			MustNotUse: []string{phpWebConfig, functionFramework, cloudFunctions},
		},
		{
			Name: "no composer.json",
			App:  "no_composer_json",
			// VersionInclusionConstraint: "< 8.2.0",
			MustNotUse: []string{composer, composerGCPBuild, composerInstall, phpWebConfig, functionFramework, cloudFunctions},
		},
		// Test that we can build an app with SDK dependencies
		{
			Name: "appengine_sdk dependencies",
			App:  "appengine_sdk",
			// VersionInclusionConstraint: "< 8.2.0",
			Env: []string{"GAE_APP_ENGINE_APIS=TRUE"},
		},
		// Test that we get a warning using SDK libraries indirectly.
		{
			Name: "appengine_sdk transitive dependencies",
			App:  "appengine_transitive_sdk",
			// VersionInclusionConstraint: "< 8.2.0",
			MustOutput: []string{"WARNING: There is an indirect dependency on App Engine APIs, but they are not enabled in app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property."},
		},
		// Test that we get a warning when GAE_APP_ENGINE_APIS is set but no lib is used.
		{
			Name: "GAE_APP_ENGINE_APIS set with no use",
			App:  "symfony",
			// VersionInclusionConstraint: "< 8.2.0",
			Env:        []string{"GAE_APP_ENGINE_APIS=TRUE"},
			MustOutput: []string{"WARNING: App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app's yaml config file."},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		if tc.Name == "" {
			tc.Name = tc.App
		}
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
