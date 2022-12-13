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
package acceptance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func getMajorMinorVersion(version string) string {
	components := strings.Split(version, ".")
	major := components[0]
	minor := components[1]

	// set patch version as 0 to not restrict the patch version for GAE
	return major + "." + minor + ".0"
}

func insertGemfileVersion(setupCtx acceptance.SetupContext) error {
	gemfilePath := filepath.Join(setupCtx.SrcDir, "Gemfile")
	gemfileOld, err := os.ReadFile(gemfilePath)
	if err != nil {
		return err
	}
	minorVersion := getMajorMinorVersion(setupCtx.RuntimeVersion)
	gemfileNew := strings.ReplaceAll(string(gemfileOld), runtimeVersionPlaceholder, minorVersion)
	return os.WriteFile(gemfilePath, []byte(gemfileNew), 0644)
}

func insertGemsRbVersion(setupCtx acceptance.SetupContext) error {
	gemsRbPath := filepath.Join(setupCtx.SrcDir, "gems.rb")
	gemsRbOld, err := os.ReadFile(gemsRbPath)
	if err != nil {
		return err
	}
	minorVersion := getMajorMinorVersion(setupCtx.RuntimeVersion)
	gemsRbNew := strings.ReplaceAll(string(gemsRbOld), runtimeVersionPlaceholder, minorVersion)
	return os.WriteFile(gemsRbPath, []byte(gemsRbNew), 0644)
}

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
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
			Name:            "rails",
			App:             "rails",
			Env:             []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp-custom.rb"},
			EnableCacheTest: true,
			MustUse:         []string{rubyRuntime, rubyRails, rubyBundle, nodeRuntime},
		},
		{
			App: "rails_inferred",
		},
		{
			App:        "rails_precompiled",
			Env:        []string{"GOOGLE_ENTRYPOINT=bundle exec bin/rails server"},
			MustNotUse: []string{nodeRuntime, nodeYarn},
		},
		{
			App:             "simple_gemfile",
			Env:             []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			EnableCacheTest: true,
			MustNotUse:      []string{nodeRuntime, nodeYarn},
		},
		{
			App:        "simple_gems",
			Env:        []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			MustNotUse: []string{nodeRuntime, nodeYarn},
		},
		{
			App:   "version_specified_gemfile",
			Env:   []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			Setup: insertGemfileVersion,
		},
		{
			App:   "version_specified_gems",
			Env:   []string{"GOOGLE_ENTRYPOINT=bundle exec ruby myapp.rb"},
			Setup: insertGemsRbVersion,
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
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

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
