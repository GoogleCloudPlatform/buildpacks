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
			Name:                       "simple dotnet 7.0 app",
			VersionInclusionConstraint: ">=7.0.0 <8.0.0",
			App:                        "simple_dotnet7",
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist:          []string{sdk},
		},
		{
			Name:                       "simple dotnet 6.0 app",
			VersionInclusionConstraint: ">=6.0.0 <7.0.0",
			App:                        "simple_dotnet6",
			MustUse:                    []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist:          []string{sdk},
		},
		{
			Name: "simple dotnet app with runtime version",
			// .NET 3.1 is not supported on Ubuntu 22.04.
			SkipStacks:        []string{"google.22", "google.min.22", "google.gae.22"},
			App:               "simple",
			Path:              "/version?want=3.1.30",
			Env:               []string{"GOOGLE_ASP_NET_CORE_VERSION=3.1.30"},
			MustUse:           []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustNotExist: []string{sdk},
		},
		{
			Name: "simple prebuilt dotnet app",
			// simple_prebuilt is a dotnet 3 app.
			VersionInclusionConstraint: "3",
			// .NET 3.1 is not supported on Ubuntu 22.04.
			SkipStacks:        []string{"google.22", "google.min.22", "google.gae.22"},
			App:               "simple_prebuilt",
			Env:               []string{"GOOGLE_ENTRYPOINT=./simple"},
			MustUse:           []string{dotnetRuntime},
			MustNotUse:        []string{dotnetSDK, dotnetPublish},
			FilesMustNotExist: []string{sdk},
		},
		{
			Name: "Dev mode",
			// Hot reloading only works on .NET 3.1, which is not supported on Ubuntu 22.04.
			SkipStacks:          []string{"google.22", "google.min.22", "google.gae.22"},
			App:                 "simple",
			Env:                 []string{"GOOGLE_DEVMODE=1", "GOOGLE_DOTNET_SDK_VERSION=3.1.x"},
			MustUse:             []string{dotnetSDK, dotnetRuntime, dotnetPublish},
			FilesMustExist:      []string{sdk, "/workspace/Startup.cs"},
			MustRebuildOnChange: "/workspace/Startup.cs",
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
