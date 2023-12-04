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
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

const (
	dotnetPublish = "google.dotnet.publish"
	dotnetRuntime = "google.dotnet.runtime"
	dotnetSDK     = "google.dotnet.sdk"
)

func TestAcceptanceDotNet(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	sdk := filepath.Join("/layers", dotnetSDK, "sdk")

	testCases := []acceptance.Test{
		{
			Name:              "app with assembly name specified",
			App:               "cs_assemblyname",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
		},
		{
			Name:              "app with local dependencies",
			App:               "cs_local_deps",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			Env:               []string{"GOOGLE_BUILDABLE=App"},
		},
		{
			Name:              "app with custom entry point (legacy)",
			App:               "cs_custom_entrypoint",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			Env:               []string{"GOOGLE_ENTRYPOINT=bin/app --flag=myflag"},
		},
		{
			Name:              "app with custom entry point (on path)",
			App:               "cs_custom_entrypoint",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			Env:               []string{"GOOGLE_ENTRYPOINT=app --flag=myflag"},
		},
		{
			Name:              "app with nested directory structure",
			App:               "cs_nested_proj",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			Env:               []string{"GOOGLE_BUILDABLE=app/app.csproj"},
		},
		{
			Name:                       "pick runtime version from buildable target",
			VersionInclusionConstraint: "3",
			App:                        "cs_local_deps_reversed",
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist:          []string{sdk},
			Env:                        []string{"GOOGLE_BUILDABLE=Function/Function.csproj"},
		},
		{
			Name:              "build with properties specified",
			App:               "cs_properties",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			Env:               []string{"GOOGLE_BUILD_ARGS=-p:Version=1.0.1.0 -p:FileVersion=1.0.1.0"},
		},
		{
			Name:              "simple dotnet app",
			App:               "simple",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
		},
		{
			Name:                       "simple prebuilt dotnet app",
			VersionInclusionConstraint: "3", // simple_prebuilt is a dotnet 3 app.
			App:                        "simple_prebuilt",
			Env:                        []string{"GOOGLE_ENTRYPOINT=./simple"},
			MustUse:                    []string{dotnetRuntime},
			MustNotUse:                 []string{dotnetSDK, dotnetPublish},
			FilesMustNotExist:          []string{sdk},
		},
		{
			Name: "Dev mode",
			// Version 6.0 of the .NET sdk made changes to watch. When running this test
			// with 6.0 watch notices that the file has changed but decides that there are
			// no Hot Swap changes to reload. Disable this test while this is being
			// investigated.
			VersionInclusionConstraint: "< 6.0",
			App:                        "simple",
			Env:                        []string{"GOOGLE_DEVMODE=1"},
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustExist:             []string{sdk, "/workspace/Startup.cs"},
			MustRebuildOnChange:        "/workspace/Startup.cs",
		},
		{
			// This is a separate test case from Dev mode above because it has a fixed runtime version.
			// Its only purpose is to test that the metadata is set correctly.
			Name: "Dev mode metadata",
			// Test is only intended to be run against a single version
			VersionInclusionConstraint: "3",
			App:                        "simple",
			Env:                        []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=3.1.409"},
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			BOM: []acceptance.BOMEntry{
				{
					Name: "devmode",
					Metadata: map[string]interface{}{
						"devmode.sync": []interface{}{
							map[string]interface{}{"dest": "/workspace", "src": "**/*.cs"},
							map[string]interface{}{"dest": "/workspace", "src": "*.csproj"},
							map[string]interface{}{"dest": "/workspace", "src": "**/*.fs"},
							map[string]interface{}{"dest": "/workspace", "src": "*.fsproj"},
							map[string]interface{}{"dest": "/workspace", "src": "**/*.vb"},
							map[string]interface{}{"dest": "/workspace", "src": "*.vbproj"},
							map[string]interface{}{"dest": "/workspace", "src": "**/*.resx"},
						},
					},
				},
			},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		tc.Setup = setupTargetFramework
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresDotNet(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name: "bad runtime version",
			// Test will produce the same results across all versions.
			VersionInclusionConstraint: "3",
			App:                        "simple",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch:                  "invalid .NET SDK version specified: improper constraint: BAD_NEWS_BEARS",
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := tc
		tc.Setup = setupTargetFramework
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
