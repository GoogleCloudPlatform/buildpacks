// Copyright 2022 Google LLC
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

package fileutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestMaybeCopyPathContents(t *testing.T) {
	testCases := []struct {
		name          string
		app           string
		copyCondition func(path string, d fs.DirEntry) (bool, error)
		wantExcluded  []string // relative path from source diriectory
	}{
		{
			name:          "copyAll",
			app:           "path_with_subdir",
			copyCondition: AllPaths,
		},
		{
			name: "skipFile",
			app:  "path_with_subdir",
			copyCondition: func(path string, d fs.DirEntry) (bool, error) {
				if filepath.Base(path) == "go.mod" {
					return false, nil
				}
				return true, nil
			},
			wantExcluded: []string{"subdir/example.com/htmlreturn/go.mod"},
		},
		{
			name: "skipDir",
			app:  "path_with_subdir",
			copyCondition: func(path string, d fs.DirEntry) (bool, error) {
				if d.IsDir() && filepath.Base(path) == "subdir" {
					return false, nil
				}
				return true, nil
			},
			wantExcluded: []string{"subdir"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			src := testdata.MustGetPath(filepath.Join("testdata", tc.app))
			if err := MaybeCopyPathContents(tmp, src, tc.copyCondition); err != nil {
				t.Fatalf("failed to copy %q to %q, error: %v", src, tmp, err)
			}

			// Don't copy the root.
			exclude := map[string]struct{}{
				".": struct{}{},
			}
			for _, excludePath := range tc.wantExcluded {
				exclude[excludePath] = struct{}{}
			}

			if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Fatalf("error walking path: %q", path)
				}

				relPath, err := filepath.Rel(src, path)
				if err != nil {
					return nil
				}

				// Check that expected paths exist.
				destPath := filepath.Join(tmp, relPath)
				if _, err := os.Stat(destPath); errors.Is(err, os.ErrNotExist) {
					if _, ok := exclude[relPath]; !ok {
						t.Errorf("file %q expected to be copied to %q, but was not", path, destPath)
					} else if d.IsDir() {
						// If a directory was excluded by the test case, stop
						// crawling the directory.
						return filepath.SkipDir
					}
				}

				return nil
			}); err != nil {
				t.Fatalf("error walking source directory %q: %v", src, err)
			}
		})
	}
}

func TestMaybeMovePathContents(t *testing.T) {
	testCases := []struct {
		name          string
		app           string
		copyCondition func(path string, d fs.DirEntry) (bool, error)
		wantExcluded  []string // relative path from source diriectory
	}{
		{
			name: "copyAll",
			app:  "path_with_subdir",
			copyCondition: func(path string, d fs.DirEntry) (bool, error) {
				return true, nil
			},
		},
		{
			name: "skipFile",
			app:  "path_with_subdir",
			copyCondition: func(path string, d fs.DirEntry) (bool, error) {
				if filepath.Base(path) == "go.mod" {
					return false, nil
				}
				return true, nil
			},
			wantExcluded: []string{"subdir/example.com/htmlreturn/go.mod"},
		},
		{
			name: "skipDir",
			app:  "path_with_subdir",
			copyCondition: func(path string, d fs.DirEntry) (bool, error) {
				if d.IsDir() && filepath.Base(path) == "subdir" {
					return false, nil
				}
				return true, nil
			},
			wantExcluded: []string{"subdir"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := testdata.MustGetPath(filepath.Join("testdata", tc.app))
			srcTmp := t.TempDir()
			destTmp := t.TempDir()
			// Copy test app into a temp directory because testdata cannot
			// be overwritten in place.
			if err := MaybeCopyPathContents(srcTmp, src, AllPaths); err != nil {
				t.Fatalf("failed to copy %q to %q, error: %v", src, srcTmp, err)
			}

			if err := MaybeMovePathContents(destTmp, srcTmp, tc.copyCondition); err != nil {
				t.Fatalf("failed to move %q to %q, error: %v", srcTmp, destTmp, err)
			}

			// Don't move the root.
			exclude := map[string]struct{}{
				".": struct{}{},
			}
			for _, excludePath := range tc.wantExcluded {
				exclude[excludePath] = struct{}{}
			}

			if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Fatalf("error walking path: %q", path)
				}

				relPath, err := filepath.Rel(src, path)
				if err != nil {
					return nil
				}

				// Check that expected paths were moved.
				destPath := filepath.Join(destTmp, relPath)
				if _, err := os.Stat(destPath); errors.Is(err, os.ErrNotExist) {
					if _, ok := exclude[relPath]; !ok {
						t.Errorf("file %q expected to be copied to %q, but was not", path, destPath)
					} else if d.IsDir() {
						// If a directory was excluded by the test case, stop
						// crawling the directory.
						return filepath.SkipDir
					}
				}

				// Check that excluded paths were NOT moved (still exist
				// in the temp source directory).
				tmpSrcPath := filepath.Join(srcTmp, relPath)
				if _, err := os.Stat(tmpSrcPath); !errors.Is(err, os.ErrNotExist) {
					if _, ok := exclude[relPath]; !ok {
						t.Errorf("file %q expected to be copied to %q, but was not", path, destPath)
					} else if d.IsDir() {
						// If a directory was excluded by the test case, stop
						// crawling the directory.
						return filepath.SkipDir
					}
				}

				return nil
			}); err != nil {
				t.Fatalf("error walking source directory %q: %v", src, err)
			}
		})
	}
}
