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
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func useBundler1(setupCtx acceptance.SetupContext) error {
	lockFile := filepath.Join(setupCtx.SrcDir, "Gemfile.lock")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		return err
	}

	re := regexp.MustCompile("(?s)BUNDLED WITH.*2.3.15")
	updated := re.ReplaceAllString(string(content), "BUNDLED WITH\n   1.17.3")
	os.WriteFile(lockFile, []byte(updated), 0644)
	return nil
}

func TestAcceptanceRuby(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "using bundler 1",
			// Bundler 1 is common with older versions of Ruby which are not supported on Ubuntu 22.04.
			SkipStacks:      []string{"google.22", "google.min.22", "google.gae.22"},
			App:             "simple",
			MustUse:         []string{rubyRuntime, rubyBundle, entrypoint},
			EnableCacheTest: true,
			Setup:           useBundler1,
			Path:            "/bundler",
			Env:             []string{"GOOGLE_RUNTIME_VERSION=2.7.*"},
			MustMatch:       "1.17.3",
		},
		{
			Name:            "entrypoint from procfile web",
			App:             "simple",
			MustUse:         []string{rubyRuntime, rubyBundle, entrypoint},
			EnableCacheTest: true,
		},
		{
			Name:       "entrypoint from procfile custom",
			App:        "simple",
			Path:       "/custom",
			Entrypoint: "custom", // Must match the non-web process in Procfile.
			MustUse:    []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "entrypoint from env",
			App:     "simple",
			Path:    "/custom",
			Env:     []string{"GOOGLE_ENTRYPOINT=ruby custom.rb"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "entrypoint with env var",
			App:     "simple",
			Path:    "/env?want=bar",
			Env:     []string{"GOOGLE_ENTRYPOINT=FOO=bar ruby main.rb"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "GOOGLE_RUNTIME_VERSION env",
			App:     "simple",
			Path:    "/version?want=3.1.0",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.1.0"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:            "rails",
			App:             "rails",
			Env:             []string{"GOOGLE_RUNTIME_VERSION=3.1.2", "GOOGLE_ENTRYPOINT=bundle exec ruby myapp-custom.rb"},
			MustUse:         []string{rubyRuntime, rubyRails, rubyBundle, entrypoint, nodeRuntime},
			EnableCacheTest: true,
		},
		{
			Name:    "rails minimal_old",
			App:     "rails_minimal",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.1.2", "GOOGLE_ENTRYPOINT=ruby bin/rails server -b 0.0.0.0 -p $PORT"},
			MustUse: []string{rubyRuntime, rubyRails, rubyBundle, entrypoint},
		},
		{
			Name:       "rails precompiled",
			App:        "rails_precompiled",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=3.1.2", "GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			MustUse:    []string{rubyRuntime, rubyBundle, entrypoint},
			MustNotUse: []string{rubyRails, nodeRuntime},
		},
		{
			Name:            "Ruby native extensions",
			App:             "native_extensions",
			MustUse:         []string{rubyRuntime, rubyBundle, entrypoint},
			EnableCacheTest: false,
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

func TestFailuresRuby(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=ruby main.rb"},
			MustMatch: "invalid Ruby Runtime version specified",
		},
		{
			Name:      "missing entrypoint",
			App:       "missing_entrypoint",
			MustMatch: `for Ruby, an entrypoint must be manually set, either with "GOOGLE_ENTRYPOINT" env var or by creating a "Procfile" file`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
