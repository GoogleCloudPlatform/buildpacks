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

package gcpbuildpack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildpack"
)

type dependencyHashInfo struct {
	lockContent string
	bpID        string
	bpVersion   string
	bpName      string
	langVersion string
}

func TestDependencyHash(t *testing.T) {
	testCases := []struct {
		name     string
		info1    dependencyHashInfo
		info2    dependencyHashInfo
		wantSame bool
	}{
		{
			name:     "same components, same hash",
			info1:    dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name", "lang-version"},
			info2:    dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name", "lang-version"},
			wantSame: true,
		},
		{
			name:  "varies by buildpack lock file",
			info1: dependencyHashInfo{"lock-content-111", "bp-id", "bp-version", "bp-name", "lang-version"},
			info2: dependencyHashInfo{"lock-content-222", "bp-id", "bp-version", "bp-name", "lang-version"},
		},
		{
			name:  "varies by buildpack ID",
			info1: dependencyHashInfo{"lock-content", "bp-id-111", "bp-version", "bp-name", "lang-version"},
			info2: dependencyHashInfo{"lock-content", "bp-id-222", "bp-version", "bp-name", "lang-version"},
		},
		{
			name:  "varies by buildpack version",
			info1: dependencyHashInfo{"lock-content", "bp-id", "bp-version-111", "bp-name", "lang-version"},
			info2: dependencyHashInfo{"lock-content", "bp-id", "bp-version-222", "bp-name", "lang-version"},
		},
		{
			name:     "does not vary by buildpack name",
			info1:    dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name-111", "lang-version"},
			info2:    dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name-222", "lang-version"},
			wantSame: true,
		},
		{
			name:  "varies by language version",
			info1: dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name", "lang-version-111"},
			info2: dependencyHashInfo{"lock-content", "bp-id", "bp-version", "bp-name", "lang-version-222"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h1 := callDependencyHash(t, tc.info1)
			h2 := callDependencyHash(t, tc.info2)
			if tc.wantSame && h1 != h2 {
				t.Errorf("hashes not the same")
			}
			if !tc.wantSame && h1 == h2 {
				t.Errorf("hashes not unique")
			}
		})
	}
}

func callDependencyHash(t *testing.T, info dependencyHashInfo) string {
	t.Helper()

	// Change current working dir to a temp dir, write lock-file.json file in the new working dir.
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd(): %v", err)
	}

	temp, err := ioutil.TempDir("", "dependency-hash-")
	if err != nil {
		t.Fatalf("ioutil.TempDir(): %v", err)
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			t.Fatalf("os.RemoveAll(%q): %v", temp, err)
		}
	}()

	if err := os.Chdir(temp); err != nil {
		t.Fatalf("os.Chdir(%q): %v", temp, err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("os.Chdir(%q): %v", oldwd, err)
		}
	}()

	if err := ioutil.WriteFile("lock-file.json", []byte(info.lockContent), 0644); err != nil {
		t.Fatalf("ioutil.WriteFile(): %v", err)
	}

	ctx := NewContext(buildpack.Info{ID: info.bpID, Version: info.bpVersion, Name: info.bpName})
	h, err := DependencyHash(ctx, info.langVersion, "lock-file.json")
	if err != nil {
		t.Fatalf("hashing %v: %v", info, err)
	}

	return h
}

func TestComputeSHA256(t *testing.T) {
	testCases := []struct {
		name       string
		boolComp   bool
		stringComp string
		intComp    int
		fileComp   string
		want       string
	}{
		{
			name:     "bool",
			boolComp: true,
			want:     "2e37d116712a5fc1780dc702a1072e173363fd222114a3c22a89e1fbb5f751ee",
		},
		{
			name:       "string",
			stringComp: "my-string",
			want:       "75e3d0ce18615f1fcca84513474b0040ec223ceac07e0079a0221a7e1704caa6",
		},
		{
			name:    "int",
			intComp: 99,
			want:    "1573010be7b56b67ec24e4a043250d367b12cc0835c6b74bdb3d380519fa790a",
		},
		{
			name:     "file contents",
			fileComp: "my-file",
			want:     "c0ab46191bac12e89ee832a06cfcf68323a50acc08bcebf462dba837ab4f93d2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var c []interface{}
			if tc.boolComp {
				c = append(c, tc.boolComp)
			}
			if tc.intComp != 0 {
				c = append(c, tc.intComp)
			}
			if tc.stringComp != "" {
				c = append(c, tc.stringComp)
			}
			if tc.fileComp != "" {
				temp, err := ioutil.TempDir("", "test-sha-")
				if err != nil {
					t.Fatalf("creating temp dir: %v", err)
				}
				defer func() {
					if err := os.RemoveAll(temp); err != nil {
						t.Fatalf("removing temp dir %q: %v", temp, err)
					}
				}()

				fname := writeFile(t, temp, tc.fileComp, "some-contents")
				c = append(c, HashFileContents(fname))
			}

			ctx := NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
			got := computeHash(t, ctx, c...)
			if got != tc.want {
				t.Errorf("computeSHA256(%v) got %q, want %q", c, got, tc.want)
			}
		})
	}
}

func TestComputeSHA256_SameFileContentsYieldsSameHash(t *testing.T) {
	temp, err := ioutil.TempDir("", "test-sha-same-contents-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			t.Fatalf("removing temp dir %q: %v", temp, err)
		}
	}()

	contents := "same-contents"
	fname1 := writeFile(t, temp, "file1", contents)
	fname2 := writeFile(t, temp, "file2", contents)

	ctx := NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
	f1 := computeHash(t, ctx, HashFileContents(fname1))
	f2 := computeHash(t, ctx, HashFileContents(fname2))
	if f1 != f2 {
		t.Errorf("file hashes do not match")
	}
}

func TestComputeSHA256_Uniqueness(t *testing.T) {
	temp, err := ioutil.TempDir("", "test-sha-uniqueness-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			t.Fatalf("removing temp dir %q: %v", temp, err)
		}
	}()

	fname1 := writeFile(t, temp, "file1", "content1")
	fname2 := writeFile(t, temp, "file2", "content2")

	testCases := [][]interface{}{
		{"string1"},
		{"string2"},
		{true},
		{false},
		{123},
		{124},
		{HashFileContents(fname1)},
		{HashFileContents(fname2)},
		{"my-string", true},
		{"my-string", false},
		{"my-other-string", true},
		{"my-string", 123},
		{"my-string", 123, true},
		{HashFileContents(fname1), true},
		{HashFileContents(fname1), false},
	}

	// Compute hash for each, remove duplicates, result must be same length as original (i.e., all unique).
	ctx := NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
	var hashes []string
	for _, tc := range testCases {
		hashes = append(hashes, computeHash(t, ctx, tc...))
	}
	cleaned := removeDuplicates(t, hashes)
	if len(cleaned) != len(hashes) {
		t.Fatalf("hashes were not unique %v", hashes)
	}
}

func writeFile(t *testing.T, tempDir, name, contents string) string {
	t.Helper()
	fullName := filepath.Join(tempDir, name)
	if err := ioutil.WriteFile(fullName, []byte(contents), 0644); err != nil {
		t.Fatalf("writing file %q: %v", fullName, err)
	}
	return fullName
}

func computeHash(t *testing.T, ctx *Context, c ...interface{}) string {
	t.Helper()
	h, err := ComputeSHA256(ctx, c...)
	if err != nil {
		t.Fatalf("computeSHA256(%v) got err=%v, want err=nil", c, err)
	}
	return h
}

func removeDuplicates(t *testing.T, original []string) []string {
	t.Helper()
	keys := make(map[string]bool)
	var result []string
	for _, entry := range original {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}
