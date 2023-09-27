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

package main

import (
	"os"
	"strconv"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "pom.xml",
			files: map[string]string{
				"pom.xml": "",
			},
			want: 0,
		},
		{
			name: ".mvn/extensions.xml",
			files: map[string]string{
				".mvn/extensions.xml": "",
			},
			want: 0,
		},
		{
			name:  "no pom.xml",
			files: map[string]string{},
			want:  100,
		},
		{
			name: "use GOOGLE_BUILDABLE",
			files: map[string]string{
				"testmodule/pom.xml": "",
			},
			env:  []string{"GOOGLE_BUILDABLE=testmodule"},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestCrLfRewrite(t *testing.T) {
	testCases := []struct {
		name          string
		inputContent  string
		expectContent string
	}{
		{
			name:          "windows-style replaced",
			inputContent:  "#!/bin/sh\r\n\r\necho Windows\r\n",
			expectContent: "#!/bin/sh\n\necho Windows\n",
		},
		{
			name:          "unix-style unmodified",
			inputContent:  "#!/bin/sh\n\necho Unix\n",
			expectContent: "#!/bin/sh\n\necho Unix\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
			if err != nil {
				t.Fatal("Cannot create temporary file", err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write([]byte(tc.inputContent))
			if err != nil {
				t.Fatal("Error writing temporary file", err)
			}
			err = tmpFile.Close()
			if err != nil {
				t.Fatal("Error closing temporary file", err)
			}

			ensureUnixLineEndings(gcp.NewContext(), tmpFile.Name())

			newContent, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatal("Error reading updated temporary file", err)
			}

			if string(newContent) != tc.expectContent {
				t.Fatal("Unexpected content '%s', want '%s'",
					strconv.QuoteToASCII(string(newContent)),
					strconv.QuoteToASCII(tc.expectContent))
			}

		})
	}
}
