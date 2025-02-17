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

package dotnet

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"google3/third_party/golang/cmp/cmpopts/cmpopts"
	"github.com/buildpacks/libcnb/v2"
)

func TestReadProjectFile(t *testing.T) {
	d, err := ioutil.TempDir("/tmp", "test-read-project-file")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(d)

	contents := `
<Project Sdk="Microsoft.NET.Sdk.Web">
	<PropertyGroup>
		<AssemblyName>Foo</AssemblyName>
		<TargetFramework>net48</TargetFramework>
		<TargetFrameworks>netcoreapp3.1;netstandard2.0</TargetFrameworks>
	</PropertyGroup>

	<ItemGroup>
		<PackageReference Include="Google.Cloud.Some.Package" Version="1.0.0" />
	</ItemGroup>
</Project>
`

	if err := ioutil.WriteFile(filepath.Join(d, "test.csproj"), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	want := Project{
		XMLName: xml.Name{Local: "Project"},
		PropertyGroups: []PropertyGroup{
			PropertyGroup{
				AssemblyName:     "Foo",
				TargetFramework:  "net48",
				TargetFrameworks: "netcoreapp3.1;netstandard2.0",
			},
		},
		ItemGroups: []ItemGroup{
			ItemGroup{
				PackageReferences: []PackageReference{
					PackageReference{
						Include: "Google.Cloud.Some.Package",
						Version: "1.0.0",
					},
				},
			},
		},
	}

	got, err := readProjectFile([]byte(contents), d)
	if err != nil {
		t.Fatalf("ReadProjectFile got error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadProjectFile\ngot %#v\nwant %#v", got, want)
	}
}

func TestRuntimeConfigJSONFiles(t *testing.T) {
	testCases := []struct {
		Name                 string
		TestDataRelativePath string
		ExpectedResult       []string
	}{
		{
			Name:                 "finds single file in root dir",
			TestDataRelativePath: "singleRtCfg",
			ExpectedResult:       []string{"singleRtCfg/my.runtimeconfig.json"},
		},
		{
			Name:                 "doesn't find recursively",
			TestDataRelativePath: "nestedRtCfg",
			ExpectedResult:       []string{},
		},
		{
			Name:                 "finds multiples in root dir",
			TestDataRelativePath: "multipleRtCfg",
			ExpectedResult:       []string{"multipleRtCfg/my.runtimeconfig.json", "multipleRtCfg/my.second.runtimeconfig.json"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rootDir := testdata.MustGetPath("testdata/runtimeconfig")
			tstDir := path.Join(rootDir, tc.TestDataRelativePath)
			files, err := RuntimeConfigJSONFiles(tstDir)
			if err != nil {
				t.Fatalf("RuntimeConfigFiles(%v) got error: %v", tstDir, err)
			}
			// the test cases are written without the full path to make writing test cases easier
			// prepend the tstDir to the relative paths to get the true expected result
			fullPathExpectedResults := make([]string, 0, len(tc.ExpectedResult))
			for _, val := range tc.ExpectedResult {
				fullPathExpectedResults = append(fullPathExpectedResults, path.Join(rootDir, val))
			}
			if !cmp.Equal(files, fullPathExpectedResults, cmpopts.SortSlices(func(a, b string) bool { return a < b })) {
				t.Errorf("RuntimeConfigFiles(%v) = %q, want %q", tstDir, files, fullPathExpectedResults)
			}
		})
	}
}

func TestReadRuntimeConfigJSON(t *testing.T) {
	path := "testdata/runtimeconfig/singleRtCfg/my.runtimeconfig.json"
	rtCfg, err := ReadRuntimeConfigJSON(testdata.MustGetPath(path))
	if err != nil {
		t.Fatalf("ReadRuntimeConfigJSON(%v) got error: %v", path, err)
	}
	expectedTFM := "netcoreapp3.1"
	if rtCfg.RuntimeOptions.TFM != expectedTFM {
		t.Errorf("unexpected tfm value: got %q, want %q", rtCfg.RuntimeOptions.TFM, expectedTFM)
	}
}

func TestGetSDKVersion(t *testing.T) {
	testCases := []struct {
		Name                 string
		SDKVersionEnvVar     string
		RuntimeVersionEnvVar string
		ApplicationRoot      string
		ExpectedResult       string
	}{
		{
			Name:                 "Should read from GOOGLE_RUNTIME_VERSION",
			RuntimeVersionEnvVar: "2.1.100",
			ApplicationRoot:      "",
			ExpectedResult:       "2.1.100",
		},
		{
			Name:             "Should read from GOOGLE_DOTNET_SDK_VERSION",
			SDKVersionEnvVar: "2.1.100",
			ApplicationRoot:  "",
			ExpectedResult:   "2.1.100",
		},
		{
			Name:                 "GOOGLE_DOTNET_SDK_VERSION takes precedence over GOOGLE_RUNTIME_VERSION",
			SDKVersionEnvVar:     "2.1.100",
			RuntimeVersionEnvVar: "3.1.100",
			ApplicationRoot:      "",
			ExpectedResult:       "2.1.100",
		},
		{
			Name:                 "Env var should take precedence over global.json",
			RuntimeVersionEnvVar: "2.1.100",
			ApplicationRoot:      testdata.MustGetPath("testdata/"),
			ExpectedResult:       "2.1.100",
		},
		{
			Name:                 "Should read from global.json",
			RuntimeVersionEnvVar: "",
			ApplicationRoot:      testdata.MustGetPath("testdata/"),
			ExpectedResult:       "3.1.100",
		},
		{
			Name:                 "Should read from global.json",
			RuntimeVersionEnvVar: "",
			ApplicationRoot:      testdata.MustGetPath("testdata/"),
			ExpectedResult:       "3.1.100",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tc.ApplicationRoot))
			if tc.SDKVersionEnvVar != "" {
				t.Setenv(envSdkVersion, tc.SDKVersionEnvVar)
			}
			if tc.RuntimeVersionEnvVar != "" {
				t.Setenv(env.RuntimeVersion, tc.RuntimeVersionEnvVar)
			}

			result, err := GetSDKVersion(ctx)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.ExpectedResult != result {
				t.Fatalf("result mismatch: got %q, want %q", result, tc.ExpectedResult)
			}
		})
	}
}

func TestGetRuntimeVersion(t *testing.T) {
	testCases := []struct {
		Name            string
		RtVersionEnvVar string
		RtCfgSearchRoot string
		ExpectedVersion string
		ExpectError     bool
		ExpectErrSubStr string
	}{
		{
			Name:            "No env var, should read from runtimeconfig.json",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/runtimeconfig/singleRtCfg/"),
			ExpectedVersion: "3.1.0",
		},
		{
			Name:            "Env var should take presidence over runtimeconfig.json",
			RtVersionEnvVar: "6.0.5",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/runtimeconfig/singleRtCfg/"),
			ExpectedVersion: "6.0.5",
		},
		{
			Name:            "No runtimeconfig.json found in root fails",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/"),
			ExpectError:     true,
		},
		{
			Name:            "Env var set, but no runtimeconfig.json found in root succeeds",
			RtVersionEnvVar: "6.0.5",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/"),
			ExpectedVersion: "6.0.5",
		},
		{
			Name:            "More than one runtimeconfig.json fails",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/runtimeconfig/multipleRtCfg"),
			ExpectError:     true,
		},
		{
			Name:            "Env var set, but more than one runtimeconfig.json succeeds",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/runtimeconfig/multipleRtCfg"),
			ExpectError:     true,
		},
		{
			Name:            "Env var not set and non-Asp runtimeconfig.json fails",
			RtCfgSearchRoot: testdata.MustGetPath("testdata/runtimeconfig/nonAspRtCfg"),
			ExpectError:     true,
			ExpectErrSubStr: "when GOOGLE_ASP_NET_CORE_VERSION absent, getting version from runtimeconfig.json failed: couldn't find runtime version for framework Microsoft.AspNetCore.App",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := gcp.NewContext()
			if tc.RtVersionEnvVar != "" {
				t.Setenv(EnvRuntimeVersion, tc.RtVersionEnvVar)
			}
			runtimeVersion, err := GetRuntimeVersion(ctx, tc.RtCfgSearchRoot)

			if tc.ExpectError == true {
				if err == nil {
					t.Fatalf("%s: got no error and expected error", tc.Name)
				} else {
					if tc.ExpectErrSubStr != "" && !strings.Contains(err.Error(), tc.ExpectErrSubStr) {
						t.Fatalf("got error message %s and expected substring in error %s", err.Error(), tc.ExpectErrSubStr)
					}
					return
				}
			}
			if err != nil {
				t.Fatalf("GetRuntimeVersion(ctx, %v) got unexpected error: %v",
					tc.RtCfgSearchRoot, err)
			}
			if tc.ExpectedVersion != runtimeVersion {
				t.Errorf("GetRuntimeVersion(ctx, %v) = %v, want %v",
					tc.RtCfgSearchRoot, runtimeVersion, tc.ExpectedVersion)
			}
		})
	}
}

func TestRequiresGlobalizationInvariant(t *testing.T) {
	testCases := []struct {
		Stack string
		Want  bool
	}{
		{
			Stack: googleMin22,
			Want:  true,
		},
		{
			Stack: "google.gae.22",
			Want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Stack, func(t *testing.T) {
			buildCtx := libcnb.BuildContext{
				StackID: tc.Stack,
			}
			ctx := gcp.NewContext(gcp.WithBuildContext(buildCtx))

			got := RequiresGlobalizationInvariant(ctx)
			if got != tc.Want {
				t.Errorf("RequiresGlobalizationInvariant(ctx) = %t, want %t", got, tc.Want)
			}
		})
	}
}
