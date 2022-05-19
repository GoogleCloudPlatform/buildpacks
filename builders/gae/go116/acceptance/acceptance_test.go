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

func TestAcceptance(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)

	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		// Test that gopath apps can rely on a vendor dependency in $GOPATH/src.
		{
			Name: "gopath GOPATH/src vendor dependency",
			App:  "gopath_main_ongopath_gopathvendordeps",
		},
		// Test that gopath apps can rely on a vendor dependency in application root.
		{
			Name: "gopath application root vendor dependency",
			App:  "gopath_main_ongopath_rootvendordeps",
		},
		// Test that we can move the main package from application root to $GOPATH/src/<path-to-main-package> where <path-to-main-package> is in 2 subdirectories.
		// go-app-stager places all packages but the main package in _gopath, so it is the responsibility of the appengine_gopath buildpack to move the main package from application root to $GOPATH/src.
		{
			Name: "gopath unstage multiple subdirectories",
			App:  "gopath_main_ongopath_subdirgopathdeps",
		},
		// Test that we can move the main package from application root to $GOPATH/src/<path-to-main-package> where <path-to-main-package> is in 1 subdirectory.
		{
			Name: "gopath unstage one subdirectory",
			App:  "gopath_main_ongopath_gopathdeps",
		},
		// Test that we can rely on gopath dependencies when the main package isn't in $GOPATH/src.
		{
			Name: "gopath dependencies with main package not in $GOPATH/src",
			App:  "gopath_main_not_ongopath_gopathdeps",
		},
		// Test that we can rely on a custom entrypoint.
		{
			Name: "gopath custom entrypoint",
			App:  "gopath_main_not_ongopath_custom_entrypoint",
			Env:  []string{"GOOGLE_ENTRYPOINT=main -passflag PASS"},
		},
		// Test that we can build a simple app.
		{
			Name: "gopath no dependencies",
			App:  "gopath",
		},

		// Test that GOOGLE_BUILDABLE takes precedence over app.yaml and go-app-stager.
		{
			Name: "gomod GOOGLE_BUILDABLE vs go-app-stager vs app.yaml main package",
			App:  "gomod_wrong_stager_mainpackage",
			Env:  []string{"GAE_YAML_MAIN=./wrongmaindir", "GOOGLE_BUILDABLE=./maindir"},
		},
		// Test that GOOGLE_BUILDABLE takes precedence over go-app-stager.
		{
			Name: "gomod GOOGLE_BUILDABLE vs go-app-stager main package",
			App:  "gomod_wrong_stager_mainpackage",
			Env:  []string{"GOOGLE_BUILDABLE=./maindir"},
		},
		// Test that GAE_YAML_MAIN takes precedence over go-app-stager.
		{
			Name: "gomod GAE_YAML_MAIN vs go-app-stager main package",
			App:  "gomod_wrong_stager_mainpackage",
			Env:  []string{"GAE_YAML_MAIN=./maindir"},
		},
		// Test that a gomod app can build main package from go-app-stager.
		{
			Name: "gomod stager main",
			App:  "gomod_stager_mainpackage",
		},
		// Test that we can build a gomod app where the prefix of the main package path chosen is the same as the module name.
		{
			Name: "gomod module matches path",
			App:  "gomod_dir_main",
			Env:  []string{"GAE_YAML_MAIN=example.com/package/maindir"},
		},
		// Test that we can a build a main package with a fully qualified package path.
		{
			Name: "gomod fully qualified package path",
			App:  "gomod_module_main",
			Env:  []string{"GAE_YAML_MAIN=example.com/package/maindir"},
		},
		// Test that we can build a gomod app with no dependencies.
		{
			Name:       "gomod no dependencies",
			App:        "gomod",
			MustOutput: []string{"go.sum not found, generating"},
		},
		{
			Name:          "gomod go.sum",
			App:           "gomod_go_sum",
			MustNotOutput: []string{"go.sum not found, generating"},
		},
		// Test that we can build an app with SDK dependencies
		{
			Name: "appengine_sdk dependencies",
			App:  "appengine_sdk",
			Env:  []string{"GAE_APP_ENGINE_APIS=TRUE"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=go116", "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}
