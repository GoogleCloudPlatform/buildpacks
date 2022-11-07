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
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceDotNet(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	sdk := filepath.Join("/layers", dotnetSDK, "sdk")

	testCases := []acceptance.Test{
		{
			Name:              "simple dotnet app",
			App:               "simple",
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
			EnableCacheTest:   true,
		},
		{
			Name:                       "simple dotnet 6.0 app",
			VersionInclusionConstraint: ">=6.0.0 <7.0.0",
			App:                        "simple_dotnet6",
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist:          []string{sdk},
		},
		{
			Name:              "simple dotnet app with runtime version",
			App:               "simple",
			Path:              "/version?want=3.1.0",
			Env:               []string{"GOOGLE_RUNTIME_VERSION=3.1.101"},
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
			BOM: []acceptance.BOMEntry{
				{
					Name: "aspnetcore",
					Metadata: map[string]interface{}{
						"version": "3.1.0", // Version specified by simple_prebuilt/simple.runtimeconfig.json.
					},
				},
			},
		},
		{
			Name:                "Dev mode",
			App:                 "simple",
			Env:                 []string{"GOOGLE_DEVMODE=1", "GOOGLE_DOTNET_SDK_VERSION=3.1.x"},
			MustUse:             []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustExist:      []string{sdk, "/workspace/Startup.cs"},
			MustRebuildOnChange: "/workspace/Startup.cs",
		},
		{
			// This is a separate test case from Dev mode above because it has a fixed runtime version.
			// Its only purpose is to test that the metadata is set correctly.
			Name:    "Dev mode metadata",
			App:     "simple",
			Env:     []string{"GOOGLE_DEVMODE=1", "GOOGLE_RUNTIME_VERSION=3.1.409"},
			MustUse: []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			BOM: []acceptance.BOMEntry{
				{
					Name: "dotnetsdk",
					Metadata: map[string]interface{}{
						"version": "3.1.409",
					},
				},
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
			Name:      "bad runtime version",
			App:       "simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "invalid .NET SDK version specified: improper constraint: BAD_NEWS_BEARS",
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
