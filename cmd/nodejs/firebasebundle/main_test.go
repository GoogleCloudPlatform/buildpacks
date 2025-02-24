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
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	bmd "github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/google/go-cmp/cmp"
)

//go:embed testdata/*
var testData embed.FS

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  int
	}{
		{
			name: "not a firebase apphosting app",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
		{
			name:  "a firebase apphosting app",
			files: map[string]string{},
			envs:  []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, tc.envs, tc.want)
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
			expectedFiles: []string{".apphosting", ".apphosting/bundle.yaml", "public", "public/test1", "test_dir", "test_dir/bundle.yaml", "test_dir/public", "test_dir/public/test1", "test_dir/test"},
			codeDir:       "CodeDir-no-bundleyaml",
		},
		{
			name: "copies ./public dir given bundle.yaml is empty",
			files: map[string]string{
				"public/test1":            "",
				".apphosting/bundle.yaml": "",
				"test_dir/test":           "",
			},
			expectedFiles: []string{".apphosting", ".apphosting/bundle.yaml", "public", "public/test1", "test_dir", "test_dir/bundle.yaml", "test_dir/public", "test_dir/public/test1", "test_dir/test"},
			codeDir:       "CodeDir-empty-bundleyaml",
		},
		{
			name: "nonexistent apphosting.yaml",
			files: map[string]string{
				".apphosting/bundle.yaml": `version: v1
runConfig:
  runCommand: node .output/server/index.mjs
outputFiles:
  serverApp:
    include:
      - .output`,
				"tobe_deleted/test": "",
				".output/test":      "",
				"test_dir/test":     "",
			},
			expectedFiles: []string{".apphosting", ".apphosting/bundle.yaml", ".output", ".output/test"},
			codeDir:       "CodeDir-required-dirs-no-apphosting",
		},
		{
			name: "nonexistent bundle.yaml",
			files: map[string]string{
				"apphosting.yaml": `outputFiles:
  serverApp:
    include:
      - keepthis_dir/keep_this_file`,
				".output/test":                "",
				"keepthis_dir/keep_this_file": "",
				"test_dir/test":               "",
			},
			expectedFiles: []string{".apphosting", ".apphosting/bundle.yaml", "apphosting.yaml", "keepthis_dir", "keepthis_dir/keep_this_file"},
			codeDir:       "CodeDir-required-dirs-no-bundle",
		},
		{
			name: "keeps only files included indicated by bundle.yaml and apphosting.yaml",
			files: map[string]string{
				"apphosting.yaml": `outputFiles:
  serverApp:
    include:
      - keepthis_dir/keep_this_file`,
				".apphosting/bundle.yaml": `version: v1
runConfig:
  runCommand: node .output/server/index.mjs
outputFiles:
  serverApp:
    include:
      - .output`,
				"tobe_deleted/test":           "",
				".output/test":                "",
				"keepthis_dir/keep_this_file": "",
				"test_dir/test":               "",
			},
			expectedFiles: []string{".apphosting", ".apphosting/bundle.yaml", ".output", ".output/test", "apphosting.yaml", "keepthis_dir", "keepthis_dir/keep_this_file"},
			codeDir:       "CodeDir-required-dirs",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithTempDir(tc.codeDir),
				bpt.WithEnvs(fmt.Sprintf("%s=test_dir", firebaseOutputBundleDir), fmt.Sprintf("%s=%sapphosting.yaml", apphostingYamlPath, filepath.Join(os.TempDir(), tc.codeDir)+"/")),
			}
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v, result: %#v", err, result)
			}

			// Compare the remaining files with the expected files
			if diff := cmp.Diff(tc.expectedFiles, existingFiles(t, filepath.Join(os.TempDir(), tc.codeDir))); diff != "" {
				t.Errorf("Unexpected files after deletion (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWalkDirStructureAndDeleteAllFilesNotIncluded(t *testing.T) {
	testCases := []struct {
		desc              string
		filesToInclude    []string
		existingFiles     []string
		expectedFiles     []string
		expectedError     bool
		apphostingInclude []string
		bundleInclude     []string
	}{
		{
			desc:              "Simple case - delete one file with bundle.yaml include",
			apphostingInclude: []string{},
			bundleInclude:     []string{"dir1/file1.txt"},
			existingFiles:     []string{"dir1/file1.txt", "dir1/file2.txt"},
			expectedFiles:     []string{"dir1", "dir1/file1.txt"},
		},
		{
			desc:              "Simple case - delete one file with apphosting.yaml include",
			apphostingInclude: []string{"dir1/file1.txt"},
			bundleInclude:     []string{},
			existingFiles:     []string{"dir1/file1.txt", "dir1/file2.txt"},
			expectedFiles:     []string{"dir1", "dir1/file1.txt"},
		},
		{
			desc:              "Keep nested files and directories with bundle.yaml include",
			apphostingInclude: []string{},
			bundleInclude:     []string{"dir1/dir2/file2.txt", "dir1/file1.txt"},
			existingFiles:     []string{"dir1/dir2/file2.txt", "dir1/file1.txt", "dir1/file3.txt"},
			expectedFiles:     []string{"dir1", "dir1/dir2", "dir1/dir2/file2.txt", "dir1/file1.txt"},
		},
		{
			desc:              "Keep nested files and directories with apphosting.yaml include",
			apphostingInclude: []string{"dir1/dir2/file2.txt", "dir1/file1.txt"},
			bundleInclude:     []string{},
			existingFiles:     []string{"dir1/dir2/file2.txt", "dir1/file1.txt", "dir1/file3.txt"},
			expectedFiles:     []string{"dir1", "dir1/dir2", "dir1/dir2/file2.txt", "dir1/file1.txt"},
		},
		{
			desc:              "Delete everything except root",
			apphostingInclude: []string{},
			bundleInclude:     []string{},
			existingFiles:     []string{"dir1/file1.txt", "dir2/file2.txt"},
			expectedFiles:     nil,
		},
		{
			desc:              "Test include all of a directory",
			apphostingInclude: []string{"dir2"},
			bundleInclude:     []string{},
			existingFiles:     []string{"dir2/file1.txt", "dir2/file2.txt", "dir3/file3.txt", "dir2/dir3/file4.txt"},
			expectedFiles:     []string{"dir2", "dir2/dir3", "dir2/dir3/file4.txt", "dir2/file1.txt", "dir2/file2.txt"},
		},
		{
			desc:              "Test include all files bundle.yaml",
			apphostingInclude: []string{"."},
			bundleInclude:     []string{},
			existingFiles:     []string{"dir2/file1.txt", "dir2/file2.txt", "dir3/file3.txt", "dir2/dir3/file4.txt"},
			expectedFiles:     []string{"dir2", "dir2/dir3", "dir2/dir3/file4.txt", "dir2/file1.txt", "dir2/file2.txt", "dir3", "dir3/file3.txt"},
		},
		{
			desc:              "Test include all files apphosting.yaml",
			apphostingInclude: []string{},
			bundleInclude:     []string{"."},
			existingFiles:     []string{"dir2/file1.txt", "dir2/file2.txt", "dir3/file3.txt", "dir2/dir3/file4.txt"},
			expectedFiles:     []string{"dir2", "dir2/dir3", "dir2/dir3/file4.txt", "dir2/file1.txt", "dir2/file2.txt", "dir3", "dir3/file3.txt"},
		},
		{
			desc:              "Test with nested directories and overlapping includes",
			apphostingInclude: []string{"dir1/subdir1/file1.txt", "dir2"},
			bundleInclude:     []string{"dir1/subdir1/file1.txt", "dir3/subdir2"},
			existingFiles:     []string{"dir1/subdir1/file1.txt", "dir1/subdir1/file2.txt", "dir2/file2.txt", "dir3/subdir1/file3.txt", "dir3/subdir2/file4.txt"},
			expectedFiles:     []string{"dir1", "dir1/subdir1", "dir1/subdir1/file1.txt", "dir2", "dir2/file2.txt", "dir3", "dir3/subdir2", "dir3/subdir2/file4.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "test-")
			if err != nil {
				t.Fatalf("Error creating temporary directory: %v", err)
			}
			defer os.RemoveAll(tempDir)
			for _, filePath := range tc.existingFiles {
				// Construct the full file path inside tempDir
				fullPath := filepath.Join(tempDir, filePath)

				// Create any necessary directories
				dir := filepath.Dir(fullPath)
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					fmt.Printf("Error creating directory: %v\n", err)
					continue // Skip to the next file
				}

				// Create the file and write content
				err = os.WriteFile(fullPath, []byte("test"), 0644)
				if err != nil {
					fmt.Printf("Error creating file: %v\n", err)
				}
			}

			bundleSchema := &bundleYaml{OutputFiles: outputFiles{ServerApp: serverApp{Include: tc.bundleInclude}}}
			apphostingSchema := &apphostingYaml{OutputFiles: outputFiles{ServerApp: serverApp{Include: tc.apphostingInclude}}}
			err = deleteFilesNotIncluded(apphostingSchema, bundleSchema, tempDir)
			if tc.expectedError {
				if err == nil {
					t.Error("Expected an error, but got nil")
				}
				return // If error is expected, skip the remaining checks
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Compare the remaining files with the expected files
			if diff := cmp.Diff(tc.expectedFiles, existingFiles(t, tempDir)); diff != "" {
				t.Errorf("Unexpected files after deletion (-want +got):\n%s", diff)
			}
		})
	}
}

// existingFiles returns all files that exist inside the given directory.
func existingFiles(t *testing.T, tempDir string) []string {
	t.Helper()
	var actualFiles []string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory structure: %w", err)
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}
		if relPath != "." { // Exclude the root directory itself
			actualFiles = append(actualFiles, relPath)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	return actualFiles
}

func TestSetMetadata(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON string
		metadataID  bmd.MetadataID
		want        bmd.MetadataValue
	}{
		{
			name:        "sets metadata correctly when detect genkit dependency",
			packageJSON: "testdata/genkit_app_package.json",
			metadataID:  bmd.IsUsingGenkit,
			want:        "^1.0.4",
		},
		{
			name:        "sets metadata correctly when detect genAI dependency",
			packageJSON: "testdata/gemini_app_package.json",
			metadataID:  bmd.IsUsingGenAI,
			want:        "^0.16.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got bmd.MetadataValue
			rawpjs, err := testData.ReadFile(tc.packageJSON)
			if err != nil {
				t.Errorf("Error reading json file %s: %v", tc.packageJSON, err)
			}
			var pjs nodejs.PackageJSON
			if err := json.Unmarshal(rawpjs, &pjs); err != nil {
				t.Errorf("Error unmarshalling json file %s: %v", tc.packageJSON, err)
			}
			setMetadata(&pjs)
			got = bmd.GlobalBuilderMetadata().GetValue(tc.metadataID)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("setMetadata() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
