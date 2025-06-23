// Copyright 2022 Google LLC
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
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:       "simple path",
			App:        "simple",
			MustMatch:  "PASS_INDEX",
			MustUse:    []string{phpRuntime, phpWebConfig, utilsNginx},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:      "entrypoint from procfile web",
			App:       "entrypoint",
			MustMatch: "PASS_INDEX",
			MustUse:   []string{phpRuntime, entrypoint, utilsNginx, phpWebConfig},
		},
		{
			Name:       "entrypoint from procfile custom",
			App:        "entrypoint",
			MustMatch:  "PASS_CUSTOM",
			Entrypoint: "custom", // Must match the non-web process in Procfile.
			MustUse:    []string{phpRuntime, entrypoint, utilsNginx, phpWebConfig},
		},
		{
			Name:      "entrypoint from env",
			App:       "simple",
			MustMatch: "PASS_INDEX",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, composerInstall, composer, entrypoint, utilsNginx, phpWebConfig},
		},
		{
			Name:       "custom path",
			App:        "simple",
			Path:       "/custom",
			MustMatch:  "PASS_CUSTOM",
			MustUse:    []string{phpRuntime, composerInstall, composer, phpWebConfig, utilsNginx},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "php ini config",
			App:        "php_ini_config",
			MustMatch:  "PASS_PHP_INI",
			MustUse:    []string{phpRuntime, phpWebConfig, utilsNginx},
			MustNotUse: []string{entrypoint},
		},
		{
			// Ubuntu 22 only supports php82 And does not support the version 7.4.27.
			SkipStacks: []string{"google.22", "google.min.22", "google.gae.22", "google.24.full", "google.24"},
			Name:       "runtime version 7.4.27",
			App:        "simple",
			Path:       "/version?want=7.4.27",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=7.4.27"},
			MustMatch:  "PASS_VERSION",
			MustUse:    []string{phpRuntime, phpWebConfig, utilsNginx},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "composer.json with gcp-build script and no dependencies",
			App:        "gcp_build_no_dependencies",
			MustMatch:  "PASS_PHP_GCP_BUILD",
			MustUse:    []string{composer, composerGCPBuild, composerInstall, phpRuntime, phpWebConfig, utilsNginx},
			MustNotUse: []string{entrypoint},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
