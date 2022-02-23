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

func TestAcceptanceRuby(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "entrypoint from procfile web",
			App:     "ruby/simple",
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:       "entrypoint from procfile custom",
			App:        "ruby/simple",
			Path:       "/custom",
			Entrypoint: "custom", // Must match the non-web process in Procfile.
			MustUse:    []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "entrypoint from env",
			App:     "ruby/simple",
			Path:    "/custom",
			Env:     []string{"GOOGLE_ENTRYPOINT=ruby custom.rb"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "entrypoint with env var",
			App:     "ruby/simple",
			Path:    "/env?want=bar",
			Env:     []string{"GOOGLE_ENTRYPOINT=FOO=bar ruby main.rb"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "runtime version from env",
			App:     "ruby/version_unlocked",
			Path:    "/version?want=2.7.5",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=2.7.5"},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:    "runtime version from Gemfile.lock",
			App:     "ruby/simple",
			Path:    "/version?want=3.0.3",
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		},
		{
			Name:       "selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=ruby", "GOOGLE_ENTRYPOINT=ruby main.rb"},
			MustUse:    []string{rubyRuntime},
			MustNotUse: []string{goRuntime, javaRuntime, nodeRuntime, pythonRuntime},
		},
		{
			Name:    "rails",
			App:     "ruby/rails",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=2.7.5", "GOOGLE_ENTRYPOINT=bundle exec ruby myapp-custom.rb"},
			MustUse: []string{rubyRuntime, rubyRails, rubyBundle, entrypoint},
		},
		{
			Name:    "rails minimal",
			App:     "ruby/rails_minimal",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.1.0", "GOOGLE_ENTRYPOINT=ruby bin/rails server -b 0.0.0.0 -p $PORT"},
			MustUse: []string{rubyRuntime, rubyRails, rubyBundle, entrypoint},
		},
		{
			Name:       "rails precompiled",
			App:        "ruby/rails_precompiled",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=2.7.5", "GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			MustUse:    []string{rubyRuntime, rubyBundle, entrypoint},
			MustNotUse: []string{rubyRails},
		},
	}
	// Tests for specific versions of Ruby available on dl.google.com.
	// Unlike with the other languages, we control the versions published to GCS.
	for _, v := range acceptance.RuntimeVersions("ruby", "3.1.0", "3.0.3", "2.7.5", "2.6.9") {
		testCases = append(testCases, acceptance.Test{
			Name:    "runtime version " + v,
			App:     "ruby/version_unlocked",
			Path:    "/version?want=" + v,
			Env:     []string{"GOOGLE_RUNTIME_VERSION=" + v},
			MustUse: []string{rubyRuntime, rubyBundle, entrypoint},
		})
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestFailuresRuby(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "ruby/version_unlocked",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=ruby main.rb"},
			MustMatch: "invalid Ruby Runtime version specified",
		},
		{
			Name:      "missing entrypoint",
			App:       "ruby/missing_entrypoint",
			MustMatch: `for Ruby, an entrypoint must be manually set, either with "GOOGLE_ENTRYPOINT" env var or by creating a "Procfile" file`,
		},
		{
			Name:                   "overrides Gemfile.lock ruby version with env",
			App:                    "ruby/simple",
			Env:                    []string{"GOOGLE_RUNTIME_VERSION=2.7.5"},
			MustMatch:              `Ruby version "3.0.3" in Gemfile.lock can't be overriden to "2.7.5" using GOOGLE_RUNTIME_VERSION environment variable`,
			SkipBuilderOutputMatch: true,
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
