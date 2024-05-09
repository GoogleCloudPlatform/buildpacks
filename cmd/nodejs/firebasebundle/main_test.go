// Copyright 2023 Google LLC
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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name:  "always opt in",
			files: map[string]string{},
			want:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}
func TestBuild(t *testing.T) {
	testCases := []struct {
		name          string
		files         map[string]string
		expectedFiles []string
		codeDir       string
	}{
		{
			name: "copies ./public dir given no bundle.yaml creates empty bundle.yaml in output dir",
			files: map[string]string{
				"public/test1":  "",
				"test_dir/test": "",
			},
			expectedFiles: []string{"test_dir/public/test1", "test_dir/bundle.yaml"},
			codeDir:       "CodeDir-no-bundleyaml",
		},
		{
			name: "copies ./public dir given bundle.yaml is empty",
			files: map[string]string{
				"public/test1":            "",
				".apphosting/bundle.yaml": "",
				"test_dir/test":           "",
			},
			expectedFiles: []string{"test_dir/public/test1", "public/test1", "test_dir/bundle.yaml", ".apphosting/bundle.yaml"},
			codeDir:       "CodeDir-empty-bundleyaml",
		},
		{
			name: "copies static files given bundle yaml contains static files",
			files: map[string]string{
				"public/test1":            "",
				"static/test1":            "",
				".apphosting/bundle.yaml": "staticAssets: [static]",
				"test_dir/test":           "",
			},
			expectedFiles: []string{"test_dir/static/test1", "test_dir/bundle.yaml", "public/test1", ".apphosting/bundle.yaml", "static/test1"},
			codeDir:       "CodeDir-staticassets-bundleyaml",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithTempDir(tc.codeDir),
				bpt.WithEnvs(fmt.Sprintf("%s=test_dir", firebaseOutputBundleDir)),
			}
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v, result: %#v", err, result)
			}
			if tc.expectedFiles != nil {
				for _, f := range tc.expectedFiles {
					_, err := os.ReadFile(filepath.Join(os.TempDir(), tc.codeDir, f))
					if err != nil {
						t.Errorf("reading file %s: %v", f, err)
					}
				}
			}
		})
	}
}
