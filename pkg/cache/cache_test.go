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

package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

func TestHashAndCheck(t *testing.T) {
	testCases := []struct {
		name         string
		prevEntries  map[string]any
		key          string
		cacheOpts    []Option
		wantHash     string
		wantCacheHit bool
	}{
		{
			name: "cacheHit",
			prevEntries: map[string]any{
				"testKey": "75e3d0ce18615f1fcca84513474b0040ec223ceac07e0079a0221a7e1704caa6",
			},
			key:          "testKey",
			cacheOpts:    []Option{WithStrings("my-string")},
			wantHash:     "75e3d0ce18615f1fcca84513474b0040ec223ceac07e0079a0221a7e1704caa6",
			wantCacheHit: true,
		},
		{
			name: "cacheMissValueChanged",
			prevEntries: map[string]any{
				"testKey": "old-value",
			},
			key:          "testKey",
			cacheOpts:    []Option{WithStrings("my-string")},
			wantHash:     "75e3d0ce18615f1fcca84513474b0040ec223ceac07e0079a0221a7e1704caa6",
			wantCacheHit: false,
		},
		{
			name:         "cacheMissNoPreviousEntry",
			key:          "testKey",
			cacheOpts:    []Option{WithStrings("my-string", "my-other-string")},
			wantHash:     "2896169f03a0b3756a77cd30c84e949e9bcde7af0869e291e06aaebbb97b6d11",
			wantCacheHit: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithBuildpackInfo(libcnb.BuildpackInfo{ID: "id", Version: "version"}))
			if tc.prevEntries == nil {
				tc.prevEntries = map[string]any{}
			}
			l := &libcnb.Layer{
				Metadata: tc.prevEntries,
			}
			hash, cached, err := HashAndCheck(ctx, l, tc.key, tc.cacheOpts...)
			if err != nil {
				t.Fatalf("HashAndCheck(%v, %v, %v) got err=%v, want err=nil", ctx, l, tc.key, err)
			}
			if cached != tc.wantCacheHit {
				t.Errorf("HashAndCheck() cache result = %t, want %t", cached, tc.wantCacheHit)
			}
			if hash != tc.wantHash {
				t.Errorf("HashAndCheck() hash result = %q, want %q", hash, tc.wantHash)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	testCases := []struct {
		key   string
		value string
	}{
		{
			key:   "testKey",
			value: "testValue",
		},
		{
			key:   "",
			value: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithBuildpackInfo(libcnb.BuildpackInfo{ID: "id", Version: "version"}))
			l := &libcnb.Layer{
				Metadata: map[string]any{},
			}
			Add(ctx, l, tc.key, tc.value)

			got := ctx.GetMetadata(l, tc.key)
			if got != tc.value {
				t.Errorf("Add() failed to add cache entry, got = %q, want %q", got, tc.value)
			}
		})
	}
}

func TestWithStrings(t *testing.T) {
	testCases := []struct {
		name    string
		strings []string
		want    string
	}{
		{
			name:    "empty",
			strings: nil,
			want:    "f464087ad8f464fb808201112072f7c7c928c00c7503b7c0166734ffb48edb63",
		},
		{
			name:    "one",
			strings: []string{"my-string"},
			want:    "75e3d0ce18615f1fcca84513474b0040ec223ceac07e0079a0221a7e1704caa6",
		},
		{
			name:    "multiple",
			strings: []string{"my-string", "my-other-string"},
			want:    "2896169f03a0b3756a77cd30c84e949e9bcde7af0869e291e06aaebbb97b6d11",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(gcp.WithBuildpackInfo(libcnb.BuildpackInfo{ID: "id", Version: "version"}))

			option := WithStrings(tc.strings...)
			got, err := hash(ctx, option)
			if err != nil {
				t.Fatalf("Hash(WithStrings(%v)) got err=%v, want err=nil", tc.strings, err)
			}
			if got != tc.want {
				t.Errorf("Hash(WithStrings(%v)) = %q, want %q", tc.strings, got, tc.want)
			}
		})
	}
}

func TestWithFiles(t *testing.T) {
	testCases := []struct {
		name   string
		option Option
		files  map[string]string
		want   string
	}{
		{
			name:  "empty",
			files: map[string]string{},
			want:  "f464087ad8f464fb808201112072f7c7c928c00c7503b7c0166734ffb48edb63",
		},
		{
			name:  "one",
			files: map[string]string{"my-file": "some-contents"},
			want:  "c0ab46191bac12e89ee832a06cfcf68323a50acc08bcebf462dba837ab4f93d2",
		},
		{
			name: "multiple same content",
			files: map[string]string{
				"my-file":       "some-contents",
				"my-other-file": "some-contents",
			},
			want: "a82a96db7573dd5a20934fc8591d1a9c5c86fed023d699f8a890d61e65050dfb",
		},
		{
			name: "multiple different content",
			files: map[string]string{
				"my-file":       "some-contents",
				"my-other-file": "some-other-contents",
			},
			want: "795ad75db1e3fe0d8eb02c4450237c66da7da0dc9f66cd207d37f292764b8ab7",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			temp, err := ioutil.TempDir("", "test-sha-")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(temp)

			var names []string
			for name, contents := range tc.files {
				fname := writeFile(t, temp, name, contents)
				names = append(names, fname)
			}
			sort.Strings(names)
			ctx := gcp.NewContext(gcp.WithBuildpackInfo(libcnb.BuildpackInfo{ID: "id", Version: "version"}))

			option := WithFiles(names...)
			got, err := hash(ctx, option)
			if err != nil {
				t.Fatalf("Hash(WithFiles(%v)) got err=%v, want err=nil", names, err)
			}
			if got != tc.want {
				t.Errorf("Hash(WithFiles(%v)) = %q, want %q", names, got, tc.want)
			}
		})
	}
}

func TestWithFilesError(t *testing.T) {
	ctx := gcp.NewContext()

	option := WithFiles("/does/not/exist")
	_, err := hash(ctx, option)
	if err == nil {
		t.Fatalf("Hash() got err=nil, want err")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Hash() error type unexpected: got %q want %q", err, os.ErrNotExist)
	}
}

func TestHash_SameFileContentsYieldsSameHash(t *testing.T) {
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

	ctx := gcp.NewContext()
	f1 := computeHash(t, ctx, WithFiles(fname1))
	f2 := computeHash(t, ctx, WithFiles(fname2))
	if f1 != f2 {
		t.Errorf("file hashes do not match")
	}
}

func TestHash_Uniqueness(t *testing.T) {
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

	testCases := [][]Option{
		{WithStrings("string1")},
		{WithStrings("string2")},
		{WithStrings("string1", "string2")},
		{WithFiles(fname1)},
		{WithFiles(fname2)},
		{WithFiles(fname1, fname2)},
		{WithStrings("my-string"), WithFiles(fname1)},
		{WithStrings("my-string"), WithFiles(fname2)},
	}

	// Compute hash for each, remove duplicates, result must be same length as original (i.e., all unique).
	ctx := gcp.NewContext()
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

func computeHash(t *testing.T, ctx *gcp.Context, opts ...Option) string {
	t.Helper()
	h, err := hash(ctx, opts...)
	if err != nil {
		t.Fatalf("Hash() got err=%v, want err=nil", err)
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
