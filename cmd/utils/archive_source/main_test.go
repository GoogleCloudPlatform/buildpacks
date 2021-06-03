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
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
			name: "clear source not set",
			want: 0,
		},
		{
			name: "clear source invalid",
			env:  []string{"GOOGLE_CLEAR_SOURCE=giraffe"},
			want: 1,
		},
		{
			name: "clear source false",
			env:  []string{"GOOGLE_CLEAR_SOURCE=false"},
			want: 0,
		},
		{
			name: "clear source true",
			env:  []string{"GOOGLE_CLEAR_SOURCE=true"},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gcp.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestArchiveSource(t *testing.T) {
	type testFile struct {
		Path    string
		Content string
		SymLink string
	}

	testCases := []struct {
		name  string
		files []testFile
	}{
		{
			name: "archive simple files",
			files: []testFile{
				testFile{Path: "index.js", Content: `console.log("Hello World");`},
				testFile{Path: "package.json", Content: "{}"},
			},
		},
		{
			name: "archive dotfiles",
			files: []testFile{
				testFile{Path: "index.js", Content: `console.log("Hello World");`},
				testFile{Path: "package.json", Content: "{}"},
				testFile{Path: ".npmrc", Content: "foo=bar"},
			},
		},
		{
			name: "archive directories",
			files: []testFile{
				testFile{Path: "src/index.js", Content: `console.log("Hello World");`},
				testFile{Path: "empty_dir"},
				testFile{Path: "package.json", Content: "{}"},
			},
		},
		{
			name: "archive symlinks",
			files: []testFile{
				testFile{Path: "src/index.js", Content: `console.log("Hello World");`},
				testFile{Path: "node_modules/.bin/start", SymLink: "src/index.js"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir, err := ioutil.TempDir("", "app")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(appDir)

			for _, f := range tc.files {
				fn := filepath.Join(appDir, f.Path)

				// Create the parent directories, if applicable.
				if dir := path.Dir(fn); dir != "" {
					if err := os.MkdirAll(dir, 0744); err != nil {
						t.Fatalf("creating directory tree %s: %v", dir, err)
					}
				}

				// File can be a normal file, symlink, or an empty directory.
				if f.Content != "" {
					if err := ioutil.WriteFile(fn, []byte(f.Content), 0644); err != nil {
						t.Fatalf("writing file %s: %v", fn, err)
					}
				} else if f.SymLink != "" {
					dn := filepath.Join(appDir, f.SymLink)
					if err := os.Symlink(dn, fn); err != nil {
						t.Fatalf("creating symlink %s to file %s: %v", fn, dn, err)
					}
				} else {
					if err := os.MkdirAll(fn, 0744); err != nil {
						t.Fatalf("creating directory %s: %v", fn, err)
					}
				}
			}

			// Archive the files in the app directory.
			srcDir, err := ioutil.TempDir("", "src")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(srcDir)

			sp := filepath.Join(srcDir, archiveName)
			archiveSource(gcp.NewContext(), sp, appDir)

			if _, err := os.Stat(sp); err != nil {
				if os.IsNotExist(err) {
					t.Fatalf("archive %s not exist", sp)
				}
			}

			// Extract the archive and compare the extracted files and the original files.
			cmd := exec.Command("tar",
				"--extract", "--preserve-permissions", "--same-owner",
				"--file="+sp,
				"--directory="+srcDir)
			if err = cmd.Run(); err != nil {
				t.Fatalf("extracting files: %v", err)
			}

			for _, f := range tc.files {
				af := filepath.Join(appDir, f.Path)
				afi, err := os.Stat(af)
				if err != nil {
					t.Fatalf("stating file %s: %v", af, err)
				}

				sf := filepath.Join(srcDir, f.Path)
				sfi, err := os.Stat(sf)
				if err != nil {
					t.Fatalf("stating file %s: %v", sf, err)
				}

				if sfi.Name() != afi.Name() {
					t.Errorf("unexpected file name, got: %v, want: %v", sfi.Name(), afi.Name())
				}

				if sfi.Size() != afi.Size() {
					t.Errorf("unexpected file size, got: %v, want: %v", sfi.Size(), afi.Size())
				}

				// File mode includes UID, GID, symlink, etc.
				if sfi.Mode() != afi.Mode() {
					t.Errorf("unexpected file mode, got: %v, want: %v", sfi.Mode(), afi.Mode())
				}

				// Compare file content if they are not directories.
				if !afi.IsDir() {
					ac, err := ioutil.ReadFile(af)
					if err != nil {
						t.Fatalf("reading file %s: %v", af, err)
					}
					sc, err := ioutil.ReadFile(sf)
					if err != nil {
						t.Fatalf("reading file %s: %v", sf, err)
					}
					if !bytes.Equal(sc, ac) {
						t.Errorf("unexpected file content, got: %v, want %v", sc, ac)
					}
				}
			}
		})
	}
}
