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

const (
	entrypoint         = "google.config.entrypoint"
	phpComposer        = "google.php.composer"
	phpComposerInstall = "google.php.composer-install"
	phpRuntime         = "google.php.runtime"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptancePHP(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:      "entrypoint from procfile web",
			App:       "php/simple",
			MustMatch: "PASS_INDEX",
			MustUse:   []string{phpRuntime, phpComposerInstall, phpComposer, entrypoint},
		},
		{
			Name:      "entrypoint from env",
			App:       "php/simple",
			MustMatch: "PASS_INDEX",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, phpComposerInstall, phpComposer, entrypoint},
		},
		{
			Name:      "entrypoint from env custom",
			App:       "php/simple",
			Path:      "/custom.php",
			MustMatch: "PASS_CUSTOM",
			Env:       []string{"GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, phpComposerInstall, phpComposer, entrypoint},
		},
		{
			Name:      "entrypoint with env var",
			App:       "php/simple",
			Path:      "/env.php?want=bar",
			MustMatch: "PASS_ENV",
			Env:       []string{"GOOGLE_ENTRYPOINT=FOO=bar php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, phpComposerInstall, phpComposer, entrypoint},
		},
		{
			Name:      "runtime version from env",
			App:       "php/simple",
			Path:      "/version.php?want=7.4.27",
			MustMatch: "PASS_VERSION",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=7.4.27", "GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustUse:   []string{phpRuntime, phpComposerInstall, phpComposer, entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailuresPHP(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "php/simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=php -S 0.0.0.0:8080"},
			MustMatch: "invalid PHP Runtime version specified",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builder, tc)
		})
	}
}
