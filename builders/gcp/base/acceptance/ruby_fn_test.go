// Copyright 2020 Google LLC
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

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "function with dependencies",
			App:     "with_dependencies",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.4.*"},
			MustUse: []string{rubyRuntime, rubyBundle, rubyFF},
		},
		{
			Name:       "function with platform-specific dependencies",
			App:        "with_platform_dependencies",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=3.1.*"},
			SkipStacks: []string{"google.24.full", "google.24"},
			MustUse:    []string{rubyRuntime, rubyBundle, rubyFF},
		},
		{
			Name:    "function with runtime env var",
			App:     "with_env_var",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.4.*"},
			RunEnv:  []string{"FOO=foo"},
			MustUse: []string{rubyRuntime, rubyBundle, rubyFF},
		},
		{
			Name:    "function in fn_source file",
			App:     "with_fn_source",
			Env:     []string{"GOOGLE_FUNCTION_SOURCE=sub_dir/custom_file.rb", "GOOGLE_RUNTIME_VERSION=3.4.*"},
			MustUse: []string{rubyRuntime, rubyBundle, rubyFF},
		},
		{
			Name:    "function using framework older than 0.7",
			App:     "with_legacy_framework",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.4.*"},
			MustUse: []string{rubyRuntime, rubyBundle, rubyFF},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Path = "/testFunction"
			tc.Env = append(tc.Env,
				"GOOGLE_FUNCTION_TARGET=testFunction",
			)

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
