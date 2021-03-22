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
	"reflect"
	"testing"

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
			gcp.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
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
				Signature: httpSignature,
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
