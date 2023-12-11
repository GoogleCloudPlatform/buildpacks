// Copyright 2021 Google LLC
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

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		stack string
		want  int
	}{
		{
			name: "with target and files [cc]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cc": "",
			},
			want: 0,
		},
		{
			name: "with target and files [cpp]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cpp": "",
			},
			want: 0,
		},
		{
			name: "with target and files [cxx]",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"main.cxx": "",
			},
			want: 0,
		},
		{
			name: "with target and CMake",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			files: map[string]string{
				"CMakeLists.txt": "",
			},
			want: 0,
		},
		{
			name: "with target no files",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			want: 100,
		},
		{
			name: "without target but with files",
			files: map[string]string{
				"main.cc": "",
			},
			want: 100,
		},
		{
			name: "without target or files",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestExtractFnInfo(t *testing.T) {
	testCases := []struct {
		fnTarget        string
		fnSignatureType string
		want            fnInfo
	}{
		{fnTarget: "HelloWorld", fnSignatureType: "",
			want: fnInfo{
				Target:    "HelloWorld",
				Namespace: "",
				ShortName: "HelloWorld",
				Signature: declarativeSignature,
			},
		},
		{fnTarget: "HelloWorld", fnSignatureType: "http",
			want: fnInfo{
				Target:    "HelloWorld",
				Namespace: "",
				ShortName: "HelloWorld",
				Signature: httpSignature,
			},
		},
		{fnTarget: "HelloWorld", fnSignatureType: "cloudevent",
			want: fnInfo{
				Target:    "HelloWorld",
				Namespace: "",
				ShortName: "HelloWorld",
				Signature: cloudEventSignature,
			},
		},
		{fnTarget: "ns0::HelloWorld", fnSignatureType: "cloudevent",
			want: fnInfo{
				Target:    "ns0::HelloWorld",
				Namespace: "ns0",
				ShortName: "HelloWorld",
				Signature: cloudEventSignature,
			},
		},
		{fnTarget: "ns0::ns1::ns2::HelloWorld", fnSignatureType: "cloudevent",
			want: fnInfo{
				Target:    "ns0::ns1::ns2::HelloWorld",
				Namespace: "ns0::ns1::ns2",
				ShortName: "HelloWorld",
				Signature: cloudEventSignature,
			},
		},
		{fnTarget: "::HelloWorld", fnSignatureType: "http",
			want: fnInfo{
				Target:    "::HelloWorld",
				Namespace: "",
				ShortName: "HelloWorld",
				Signature: httpSignature,
			},
		},
		{fnTarget: "::ns0::HelloWorld", fnSignatureType: "http",
			want: fnInfo{
				Target:    "::ns0::HelloWorld",
				Namespace: "::ns0",
				ShortName: "HelloWorld",
				Signature: httpSignature,
			},
		},
	}
	for _, tc := range testCases {
		got := extractFnInfo(tc.fnTarget, tc.fnSignatureType)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("unexpected output from extractFnInfo(%s, %s), got=%v, want=%v", tc.fnTarget, tc.fnSignatureType, got, tc.want)
		}
	}
}

func TestPopulateMainLayer(t *testing.T) {
	const generatedFileContents = "// test-only: generated"
	const converterFileContents = "// test-only: converter"
	testCases := []struct {
		name              string
		sourceFiles       []string
		vcpkgJSONContents string
	}{
		{
			name:              "cpp-test-both",
			sourceFiles:       []string{"CMakeLists.txt", "vcpkg.json"},
			vcpkgJSONContents: generatedFileContents,
		},
		{
			name:              "cpp-test-no-vcpkg-json",
			sourceFiles:       []string{"CMakeLists.txt"},
			vcpkgJSONContents: converterFileContents,
		},
		{
			name:              "cpp-test-no-CMakeLists-txt",
			sourceFiles:       []string{"vcpkg.json"},
			vcpkgJSONContents: generatedFileContents,
		},
		{
			name:              "cpp-test-no-support-files",
			sourceFiles:       []string{},
			vcpkgJSONContents: converterFileContents,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", tc.name)
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			fakeBuildpackRoot := filepath.Join(tmpDir, "fake-buildpack-root")
			fakeMain := filepath.Join(tmpDir, "fake-main")
			fakeApp := filepath.Join(tmpDir, "fake-app")
			for _, p := range []string{fakeBuildpackRoot, fakeMain, fakeApp} {
				if err := os.Mkdir(p, 0755); err != nil {
					t.Fatalf("creating directory structure (path=%s): %v", p, err)
				}
			}
			converter := filepath.Join(fakeBuildpackRoot, "converter")
			if err := os.Mkdir(converter, 0755); err != nil {
				t.Fatalf("creating directory structure (path=%s): %v", converter, err)
			}
			ctx := gcp.NewContext(gcp.WithApplicationRoot(fakeApp))

			for _, name := range []string{"CMakeLists.txt", "vcpkg.json"} {
				path := filepath.Join(converter, name)
				if os.WriteFile(path, []byte(converterFileContents), 0644); err != nil {
					t.Fatalf("writing fake C++ support file (path=%s): %v", path, err)
				}
			}
			for _, name := range tc.sourceFiles {
				path := filepath.Join(fakeApp, name)
				if os.WriteFile(path, []byte(generatedFileContents), 0644); err != nil {
					t.Fatalf("writing test C++ build file (name=%s): %v", name, err)
				}
			}
			if err := createMainCppSupportFiles(ctx, fakeMain, fakeBuildpackRoot); err != nil {
				t.Fatalf("creating support files in main layer: %v", err)
			}
			vcpkgContents, err := os.ReadFile(filepath.Join(fakeMain, "vcpkg.json"))
			if err != nil {
				t.Fatalf("reading vcpkg.json from main layer: %v", err)
			}
			if string(vcpkgContents) != tc.vcpkgJSONContents {
				t.Errorf("mismatched contents in vcpkg.json, got=%s, want=%s", string(vcpkgContents), tc.vcpkgJSONContents)
			}

			cmakeContents, err := os.ReadFile(filepath.Join(fakeMain, "CMakeLists.txt"))
			if err != nil {
				t.Fatalf("reading CMakeLists.txt from main layer: %v", err)
			}
			if string(cmakeContents) != converterFileContents {
				t.Errorf("mismatched contents in CMakeLists.txt, got=%s, want=%s", string(cmakeContents), converterFileContents)
			}
		})
	}
}
