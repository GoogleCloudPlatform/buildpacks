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
)

func TestExtractPackageNameFromDir(t *testing.T) {
	tcs := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name:  "one package",
			files: map[string]string{"foo.go": "package foo"},
			want:  "foo",
		},
		{
			name: "one package with two files",
			files: map[string]string{
				"foo.go":  "package foo",
				"foo2.go": "package foo",
			},
			want: "foo",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "golang_bp_test")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Fatalf("removing temp dir: %v", err)
				}
			}()

			for f, c := range tc.files {
				if err := ioutil.WriteFile(filepath.Join(dir, f), []byte(c), 0644); err != nil {
					t.Fatalf("writing file %s: %v", f, err)
				}
			}

			pkg, err := extract(dir)
			if err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}
			if pkg != tc.want {
				t.Errorf("incorrect package: got %v, want %v", pkg, tc.want)
			}
		})
	}
}

func TestExtractPackageNameFromDirErrors(t *testing.T) {
	tcs := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "two packages",
			files: map[string]string{
				"foo.go": "package foo",
				"bar.go": "package bar",
			},
		},
		{
			name: "no packages",
		},
		{
			name: "bad file",
			files: map[string]string{
				"foo.go": "not a go file",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "golang_bp_test")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Fatalf("removing temp dir: %v", err)
				}
			}()

			for f, c := range tc.files {
				if err := ioutil.WriteFile(filepath.Join(dir, f), []byte(c), 0644); err != nil {
					t.Fatalf("writing file %s: %v", f, err)
				}
			}

			_, err = extract(dir)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
