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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	npm          = "google.nodejs.npm"
	npmGCPBuild  = "google.nodejs.npm-gcp-build"
	yarn         = "google.nodejs.yarn"
	yarnGCPBuild = "google.nodejs.yarn-gcp-build"
	appEngine    = "google.nodejs.appengine"
)

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			App:        "no_package_json",
			MustUse:    []string{appEngine},
			MustNotUse: []string{npm, yarn},
		},
		{
			App:        "package_json_no_dependencies",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "package_json",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "package_lock",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "yarn_lock",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			App:        "yarn_lock_before_package_lock",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			App:        "postinstall_npm",
			MustUse:    []string{npm},
			MustNotUse: []string{npmGCPBuild, yarn, yarnGCPBuild},
			MustNotOutputCached: []string{
				"removing existing node_modules/ before installation",
				"added 1 package",
			},
		},
		{
			App:              "postinstall_yarn",
			MustUse:          []string{yarn},
			MustNotUse:       []string{npmGCPBuild, npm, yarnGCPBuild},
			MustOutputCached: []string{"Already up-to-date."},
		},
		{
			App:        "gcp_build_npm",
			MustUse:    []string{npmGCPBuild},
			MustNotUse: []string{yarnGCPBuild},
		},
		{
			App:        "gcp_build_yarn",
			MustUse:    []string{yarnGCPBuild},
			MustNotUse: []string{npmGCPBuild},
		},
		{
			App:        "gcp_build_npm_no_dependencies",
			MustUse:    []string{npmGCPBuild},
			MustNotUse: []string{yarnGCPBuild},
		},
		{
			App:        "gcp_build_yarn_no_dependencies",
			MustUse:    []string{yarnGCPBuild},
			MustNotUse: []string{npmGCPBuild},
		},
		{
			App:        "local_dependency_npm",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "local_dependency_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			App:        "dev_dependency_npm",
			Env:        []string{"NODE_ENV=development"},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "dev_dependency_yarn",
			Env:        []string{"NODE_ENV=development"},
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			App:        "node_modules_bin_npm",
			Path:       "/index.txt",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "node_modules/.bin custom entrypoint",
			App:        "node_modules_bin_npm",
			Path:       "/index.txt",
			Env:        []string{"GOOGLE_ENTRYPOINT=http-server -p $PORT"},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			App:        "node_modules_bin_yarn",
			Path:       "/index.txt",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "node_modules/.bin yarn custom entrypoint",
			App:        "node_modules_bin_yarn",
			Path:       "/index.txt",
			Env:        []string{"GOOGLE_ENTRYPOINT=http-server -p $PORT"},
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			App:        "custom_entrypoint",
			Env:        []string{"GOOGLE_ENTRYPOINT=node custom.js"},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
	}
	for _, tc := range testCases {
		tc := tc
		if tc.Name == "" {
			tc.Name = tc.App
		}
		tc.Env = append(tc.Env, "GOOGLE_RUNTIME=nodejs12")

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
