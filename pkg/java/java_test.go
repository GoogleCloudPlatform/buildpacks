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

package java

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
	"github.com/buildpack/libbuildpack/layers"
)

func TestHasMainTrue(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name: "simple case",
			manifestContents: `Main-Class: test
another: example`,
		},
		{
			name: "with line continuation",
			// The manifest spec states that a continuation line must end with a trailing space.
			manifestContents: `simple: example 
 wrapping
Main-Class: example`,
		},
		{
			name: "main-class with line continuation",
			manifestContents: `simple: example 
 wrapping
Main-Class: example 
 line wrap
New-Entry: example`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.manifestContents)

			if !hasMain(r) {
				t.Errorf("hasMain() returned false, wanted true")
			}
		})
	}
}

func TestHasMainFalse(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name: "no main class entry",
			manifestContents: `Not-Main-Class: test
another: example`,
		},
		{
			name: "main class with preceding space",
			manifestContents: `simple: example
 wrapping
 Main-Class: example`,
		},
		{
			name:             "main class with no entry",
			manifestContents: `Main-Class: `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.manifestContents)

			if hasMain(r) {
				t.Errorf("hasMain() returned true, wanted false")
			}
		})
	}
}

func TestMainFromManifest(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
		want             string
	}{
		{
			name:             "simple case",
			manifestContents: `Main-Class: test`,
			want:             "test",
		},
		{
			name: "2 line case",
			manifestContents: `ex: example
Main-Class: test`,
			want: "test",
		},
		{
			name: "3 line case",
			manifestContents: `ex: example
Main-Class: test
another: example`,
			want: "test",
		},
		{
			name: "3 line with trailing space case",
			manifestContents: `ex: example 
Main-Class: test 
another: example`,
			want: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mfPath := setupTestManifest(t, []byte(tc.manifestContents))
			ctx := gcp.NewContextForTests(buildpack.Info{}, "")
			got, err := MainFromManifest(ctx, mfPath)
			if err != nil {
				t.Errorf("MainFromMainfest() errored: %v", err)
			}

			if got != tc.want {
				t.Errorf("MainFromMainfest() returned %s, wanted %s", got, tc.want)
			}
		})
	}
}

func TestMainFromManifestFail(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name:             "empty manifest",
			manifestContents: ``,
		},
		{
			name:             "no main-class entry found",
			manifestContents: `key: value`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mfPath := setupTestManifest(t, []byte(tc.manifestContents))
			ctx := gcp.NewContextForTests(buildpack.Info{}, "")
			_, err := MainFromManifest(ctx, mfPath)
			if err == nil {
				t.Error("MainFromMainfest() did not error as expected")
			}
		})
	}
}

func TestCheckCacheNewDateMiss(t *testing.T) {
	testCases := []struct {
		name            string
		expiryTimestamp string
	}{
		{
			name:            "empty string",
			expiryTimestamp: "",
		},
		{
			name:            "invalid format",
			expiryTimestamp: "invalid format",
		},
		{
			name:            "old expiry date",
			expiryTimestamp: time.Now().Truncate(repoExpiration).Format(dateFormat),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
			repoMeta := &RepoMetadata{
				ExpiryTimestamp: tc.expiryTimestamp,
			}

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)

			CheckCacheExpiration(ctx, repoMeta, m2CachedRepo)
			if repoMeta.ExpiryTimestamp == tc.expiryTimestamp {
				t.Errorf("checkCacheExpiration() did not set new date when expected to with ExpiryTimestamp: %q", repoMeta.ExpiryTimestamp)
			}
			if ctx.FileExists(testFilePath) {
				ctx.RemoveAll(testFilePath)
				t.Errorf("checkCacheExpiration() did not clear layer")
			}
		})
	}
}

func TestCheckCacheNewDateHit(t *testing.T) {
	testCases := []struct {
		name            string
		expiryTimestamp string
	}{
		{
			name:            "expiry date in the future",
			expiryTimestamp: time.Now().Add(repoExpiration).Format(dateFormat),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
			repoMeta := &RepoMetadata{
				ExpiryTimestamp: tc.expiryTimestamp,
			}

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)

			CheckCacheExpiration(ctx, repoMeta, m2CachedRepo)
			if repoMeta.ExpiryTimestamp != tc.expiryTimestamp {
				t.Errorf("checkCacheExpiration() set new date when expected not to with ExpiryTimestamp: %q", repoMeta.ExpiryTimestamp)
			}
			if !ctx.FileExists(testFilePath) {
				t.Errorf("checkCacheExpiration() cleared layer")
			}
			// Clean up layer for next test case.
			ctx.RemoveAll(testFilePath)
		})
	}
}

func setupTestLayer(t *testing.T, ctx *gcp.Context) (string, *layers.Layer) {
	testLayerRoot, err := ioutil.TempDir("", "test-layer-")
	if err != nil {
		t.Fatalf("Creating temp directory: %v", err)
	}
	testFilePath := filepath.Join(testLayerRoot, "testfile")
	ctx.CreateFile(testFilePath)
	m2CachedRepo := &layers.Layer{
		Root: testLayerRoot,
	}
	return testFilePath, m2CachedRepo
}

func setupTestManifest(t *testing.T, mfContent []byte) string {
	t.Helper()
	tDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	mfPath := filepath.Join(tDir, "TEST.MF")
	err = ioutil.WriteFile(mfPath, mfContent, 0644)
	if err != nil {
		t.Fatalf("writing to file %s: %v", mfPath, err)
	}
	return mfPath
}
