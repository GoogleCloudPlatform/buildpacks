// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	composer         = "google.php.composer"
	composerGCPBuild = "google.php.composer-gcp-build"
)

func TestAcceptance(t *testing.T) {

	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "symfony app",
			App:        "symfony",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
		},
		{
			Name:       "composer.json without dependencies",
			App:        "composer_json_no_dependencies",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
		},
		{
			Name:       "composer.lock respected",
			App:        "composer_lock",
			MustUse:    []string{composer},
			MustNotUse: []string{composerGCPBuild},
		},
		{
			Name:    "composer.json with gcp-build script and no dependencies",
			App:     "gcp_build_no_dependencies",
			MustUse: []string{composer, composerGCPBuild},
		},
		{
			Name:       "no composer.json",
			App:        "no_composer_json",
			MustNotUse: []string{composer, composerGCPBuild},
		},
		// Test that we can build an app with SDK dependencies
		{
			Name: "appengine_sdk dependencies",
			App:  "appengine_sdk",
			Env:  []string{"GAE_APP_ENGINE_APIS=TRUE"},
		},
		// Test that we get a warning using SDK libraries indirectly.
		{
			Name:       "appengine_sdk transitive dependencies",
			App:        "appengine_transitive_sdk",
			MustOutput: []string{"WARNING: There is an indirect dependency on App Engine APIs, but they are not enabled in app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property."},
		},
		// Test that we get a warning when GAE_APP_ENGINE_APIS is set but no lib is used.
		{
			Name:       "GAE_APP_ENGINE_APIS set with no use",
			App:        "symfony",
			Env:        []string{"GAE_APP_ENGINE_APIS=TRUE"},
			MustOutput: []string{"WARNING: App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app's yaml config file."},
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.Name == "" {
			tc.Name = tc.App
		}
		tc.Env = append(tc.Env, "GOOGLE_RUNTIME=php81")

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
