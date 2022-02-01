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
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
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
			Name:                 "single file in same directory",
			TestDataRelativePath: "subdir",
			ExpectedResult:       []string{"subdir/my.runtimeconfig.json"},
		},
		{
			Name:                 "single file in sub-directory",
			TestDataRelativePath: "another_dir",
			ExpectedResult:       []string{"another_dir/with_subfolder/another.runtimeconfig.json"},
		},
		{
			Name:                 "multiple entries",
			TestDataRelativePath: "",
			ExpectedResult: []string{
				"another_dir/with_subfolder/another.runtimeconfig.json",
				"runtimeconfig.json/test.runtimeconfig.json",
				"subdir/my.runtimeconfig.json",
			},
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
			if !reflect.DeepEqual(files, fullPathExpectedResults) {
				t.Errorf("RuntimeConfigFiles(%v) = %q, want %q", tstDir, files, fullPathExpectedResults)
			}
		})
	}
}

func TestReadRuntimeConfigJSON(t *testing.T) {
	path := "testdata/runtimeconfig/subdir/my.runtimeconfig.json"
	rtCfg, err := ReadRuntimeConfigJSON(testdata.MustGetPath(path))
	if err != nil {
		t.Fatalf("ReadRuntimeConfigJSON(%v) got error: %v", path, err)
	}
	expectedTFM := "netcoreapp3.1"
	if rtCfg.RuntimeOptions.TFM != expectedTFM {
		t.Errorf("unexpected tfm value: got %q, want %q", rtCfg.RuntimeOptions.TFM, expectedTFM)
	}
}

func TestGetSDKorRuntimeVersion(t *testing.T) {
	testCases := []struct {
		Name                 string
		RuntimeVersionEnvVar string
		ApplicationRoot      string
		ExpectedResult       string
	}{
		{
			Name:                 "Should read from env var",
			RuntimeVersionEnvVar: "2.1.100",
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
			os.Setenv("GOOGLE_RUNTIME_VERSION", tc.RuntimeVersionEnvVar)

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
