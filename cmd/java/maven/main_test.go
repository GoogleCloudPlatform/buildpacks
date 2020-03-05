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
	"path/filepath"
	"testing"
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
	"github.com/buildpack/libbuildpack/layers"
)

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
			repoMeta := &repoMetadata{
				ExpiryTimestamp: tc.expiryTimestamp,
			}

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)

			checkCacheExpiration(ctx, repoMeta, m2CachedRepo)
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
			repoMeta := &repoMetadata{
				ExpiryTimestamp: tc.expiryTimestamp,
			}

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)

			checkCacheExpiration(ctx, repoMeta, m2CachedRepo)
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
