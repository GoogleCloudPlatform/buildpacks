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
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)
	testCases := []acceptance.Test{

		{
			App:             "rack",
			Env:             []string{"GOOGLE_ENTRYPOINT=bundle exec rackup -p $PORT config-custom.ru"},
			EnableCacheTest: true,
		},
		{
			App: "rack_inferred",
		},
		{
			App:             "rails",
			Env:             []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp-custom.rb"},
			EnableCacheTest: true,
		},
		{
			App: "rails_inferred",
		},
		{
			App: "rails_precompiled",
			Env: []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
		},
		{
			App:             "simple_gemfile",
			Env:             []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			EnableCacheTest: true,
		},
		{
			App: "simple_gems",
			Env: []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
		},
		{
			App: "version_specified_gemfile_30",
			Env: []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
		},
		{
			App: "version_specified_gems_30",
			Env: []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=ruby30", "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			App:       "fail_cannot_infer_entrypoint",
			MustMatch: "unable to infer entrypoint",
		},
		{
			App:       "fail_version_pinned_gemfile",
			Env:       []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			MustMatch: "Your Gemfile cannot restrict the Ruby version",
		},
		{
			App:       "fail_version_pinned_gems",
			Env:       []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			MustMatch: "Your gems.rb cannot restrict the Ruby version",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=ruby30", "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
