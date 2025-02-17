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
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

func TestFindManifestValueFromJar(t *testing.T) {
	testCases := []struct {
		name             string
		key              string
		manifestContents string
		want             string
	}{
		{
			name: "simple case",
			key:  "Main-Class",
			manifestContents: `Main-Class: test
another: example`,
			want: "test",
		},
		{
			name: "with line continuation",
			key:  "Start-Class",
			// The manifest spec states that a continuation line must end with a trailing space.
			manifestContents: `simple: example 
 wrapping
Start-Class: example`,
			want: "example",
		},
		{
			name: "main-class with line continuation",
			key:  "Main-Class",
			manifestContents: `simple: example 
 wrapping
Main-Class: example 
 line wrap
New-Entry: example`,
			want: "example",
		},
		{
			name: "no start class entry",
			key:  "Start-Class",
			manifestContents: `Not-Start-Class: test
another: example`,
			want: "",
		},
		{
			name: "main class with preceding space",
			key:  "Main-Class",
			manifestContents: `simple: example
 wrapping
 Main-Class: example`,
			want: "",
		},
		{
			name:             "start class with no entry",
			key:              "Start-Class",
			manifestContents: `Start-Class: `,
			want:             "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jarPath := setupTestJar(t, []byte(tc.manifestContents))
			got, err := FindManifestValueFromJar(jarPath, tc.key)
			if err != nil {
				t.Errorf("FindManifestValueFromJar() errored: %v", err)
			}

			if got != tc.want {
				t.Errorf("FindManifestValueFromJar()=%q want %q", got, tc.want)
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
			ctx := gcp.NewContext()
			got, err := MainFromManifest(ctx, mfPath)
			if err != nil {
				t.Errorf("MainFromMainfest() errored: %v", err)
			}

			if got != tc.want {
				t.Errorf("MainFromMainfest()=%q want %q", got, tc.want)
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
			ctx := gcp.NewContext()
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
			ctx := gcp.NewContext()

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)
			ctx.SetMetadata(m2CachedRepo, "expiry_timestamp", tc.expiryTimestamp)

			if err := CheckCacheExpiration(ctx, m2CachedRepo); err != nil {
				t.Fatalf("CheckCacheExpiration() unexpected error = %q", err.Error())
			}
			metaExpiry := ctx.GetMetadata(m2CachedRepo, "expiry_timestamp")
			if metaExpiry == tc.expiryTimestamp {
				t.Errorf("checkCacheExpiration() did not set new date when expected to with ExpiryTimestamp: %q", metaExpiry)
			}
			testFilePathExists, err := ctx.FileExists(testFilePath)
			if err != nil {
				t.Fatalf("Error checking if file exists: %v", err)
			}
			if testFilePathExists {
				if err := os.RemoveAll(testFilePath); err != nil {
					t.Errorf("error cleaning up: %v", err)
				}
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
			ctx := gcp.NewContext()
			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)
			ctx.SetMetadata(m2CachedRepo, "expiry_timestamp", tc.expiryTimestamp)

			if err := CheckCacheExpiration(ctx, m2CachedRepo); err != nil {
				t.Fatalf("CheckCacheExpiration() unexpected error = %q", err.Error())
			}
			if got, want := ctx.GetMetadata(m2CachedRepo, "expiry_timestamp"), tc.expiryTimestamp; got != want {
				t.Errorf("checkCacheExpiration() set new date when expected not to with ExpiryTimestamp: %q", got)
			}
			if _, err := os.Stat(testFilePath); err != nil {
				t.Errorf("checkCacheExpiration() cleared layer")
			}
			// Clean up layer for next test case.
			if err := os.RemoveAll(testFilePath); err != nil {
				t.Fatalf("error cleaning up: %v", err)
			}
		})
	}
}

func setupTestLayer(t *testing.T, ctx *gcp.Context) (string, *libcnb.Layer) {
	t.Helper()
	testLayerRoot := t.TempDir()
	testFilePath := filepath.Join(testLayerRoot, "testfile")
	_, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("error creating %s: %v", testFilePath, err)
	}
	m2CachedRepo := &libcnb.Layer{
		Path:     testLayerRoot,
		Metadata: map[string]interface{}{},
	}
	return testFilePath, m2CachedRepo
}

func setupTestManifest(t *testing.T, mfContent []byte) string {
	t.Helper()
	mfPath := filepath.Join(t.TempDir(), "TEST.MF")
	if err := ioutil.WriteFile(mfPath, mfContent, 0644); err != nil {
		t.Fatalf("writing to file %s: %v", mfPath, err)
	}
	return mfPath
}

func setupTestJar(t *testing.T, mfContent []byte) string {
	t.Helper()
	var buff bytes.Buffer
	w := zip.NewWriter(&buff)
	defer w.Close()
	f, err := w.Create(filepath.Join("META-INF", "MANIFEST.MF"))
	if err != nil {
		t.Fatalf("creating zip entry: %v", err)
	}
	for i := 0; i < len(mfContent); {
		n, err := f.Write(mfContent)
		if err != nil {
			t.Fatalf("writing bytes: %v", err)
		}
		i += n
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}

	jarPath := filepath.Join(t.TempDir(), "test.jar")
	if err := ioutil.WriteFile(jarPath, buff.Bytes(), 0644); err != nil {
		t.Fatalf("writing to file %s: %v", jarPath, err)
	}
	return jarPath
}
