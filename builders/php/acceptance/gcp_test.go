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
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:      "entrypoint from procfile web",
			App:       "simple",
			MustMatch: "PASS_INDEX",
			MustUse:   []string{phpRuntime, composerInstall, composer, entrypoint},
		},
		{
			Name:      "entrypoint from env",
			App:       "simple",
			MustMatch: "PASS_INDEX",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, composerInstall, composer, entrypoint},
		},
		{
			Name:      "entrypoint from env custom",
			App:       "simple",
			Path:      "/custom.php",
			MustMatch: "PASS_CUSTOM",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, composerInstall, composer, entrypoint},
		},
		{
			Name:      "entrypoint with env var",
			App:       "simple",
			Path:      "/env.php?want=bar",
			MustMatch: "PASS_ENV",
			Env:       []string{"GOOGLE_ENTRYPOINT=FOO=bar php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, composerInstall, composer, entrypoint},
		},
		{
			Name:      "php ini config",
			App:       "php_ini_config",
			MustMatch: "PASS_PHP_INI",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}
