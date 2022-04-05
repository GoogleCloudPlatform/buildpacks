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
	"io/ioutil"
	"os"
	"path/filepath"
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
			name: "go.mod and buildable undefined",
			files: map[string]string{
				"go.mod": "",
			},
			env:  []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want: 0,
		},
		{
			name: "go.mod and buildable undefined, no target_platform",
			files: map[string]string{
				"go.mod": "",
			},
			env:  []string{},
			want: 100,
		},
		{
			name:  "no go.mod",
			files: map[string]string{},
			env:   []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want:  100,
		},
		{
			name: "buildable defined",
			files: map[string]string{
				"go.mod": "",
			},
			env: []string{
				"GOOGLE_BUILDABLE=./main",
				"X_GOOGLE_TARGET_PLATFORM=gae",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestMainPath(t *testing.T) {
	testCases := []struct {
		name               string
		stagerFileContents string
		gaeMainEnv         string
		want               string
	}{
		{
			name:       "no stagerfile and an empty env var",
			gaeMainEnv: "",
			want:       "",
		},
		{
			name:               "stagerfile with main directory and an empty env var",
			stagerFileContents: "maindir",
			gaeMainEnv:         "",
			want:               "maindir",
		},
		{
			name:       "no stagerfile and a non-empty env var",
			gaeMainEnv: "anothermaindir",
			want:       "anothermaindir",
		},
		{
			name:               "stagerfile with main directory and a non-empty env var",
			stagerFileContents: "maindir",
			gaeMainEnv:         "anothermaindir",
			want:               "anothermaindir",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "TestMainPath-")
			if err != nil {
				t.Fatalf("Creating temporary directory: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatalf("Unable to remove test directory %q", dir)
				}
			}()

			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))

			if tc.stagerFileContents != "" {
				if err = ioutil.WriteFile(filepath.Join(dir, "_main-package-path"), []byte(tc.stagerFileContents), 0755); err != nil {
					t.Fatalf("Creating file in temporary directory: %v", err)
				}
			}

			oldEnv := os.Getenv("GAE_YAML_MAIN")
			if err = os.Setenv("GAE_YAML_MAIN", tc.gaeMainEnv); err != nil {
				t.Fatalf("Setting environment variable GAE_YAML_MAIN to %q", tc.gaeMainEnv)
			}
			defer func() {
				if err := os.Setenv("GAE_YAML_MAIN", oldEnv); err != nil {
					t.Fatalf("Unable to reset the env var GAE_YAML_MAIN to %q", oldEnv)
				}
			}()

			got, err := mainPath(ctx)
			if err != nil {
				t.Fatalf("mainPath() failed unexpectedly; err=%s", err)
			}
			if got != tc.want {
				t.Errorf("mainPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCleanMainPathNoError(t *testing.T) {
	testCases := []struct {
		str  string
		want string
	}{
		{
			str:  ".",
			want: ".",
		},
		{
			str:  "   .   ",
			want: ".",
		},
		{
			str:  "./dir/..",
			want: ".",
		},
		{
			str:  "./dir1/dir2/..",
			want: "dir1",
		},
		{
			str:  "./dir1///dir2",
			want: "dir1/dir2",
		},
		{
			str:  "dir1///dir2",
			want: "dir1/dir2",
		},
		{
			str:  "dir1",
			want: "dir1",
		},
		{
			str:  "dir1/../dir2",
			want: "dir2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.str, func(t *testing.T) {
			if got, err := cleanMainPath(tc.str); err != nil {
				t.Errorf("cleanMainPath(%q) returns error: %v", tc.str, err)
			} else if got != tc.want {
				t.Errorf("cleanMainPath(%q) = %q, want %q", tc.str, got, tc.want)
			}
		})
	}
}

func TestCleanMainPathWantError(t *testing.T) {
	testCases := []string{
		"/.",
		"/somedir",
		"./..",
		"../dir1",
		"../dir1/dir2",
		"dir1/../../dir2",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			if got, err := cleanMainPath(tc); err == nil {
				t.Errorf("cleanMainPath(%q) = %q, expected error", tc, got)
			}
		})
	}
}
