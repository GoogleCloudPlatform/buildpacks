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

package runtime

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/buildpacks/libcnb"
)

func TestInstallDartSDK(t *testing.T) {
	testCases := []struct {
		name         string
		httpStatus   int
		responseFile string
		wantFile     string
		wantError    bool
	}{
		{
			name:         "successful install",
			responseFile: "testdata/dummy-dart-sdk.zip",
			wantFile:     "lib/foo.txt",
		},
		{
			name:       "invalid version",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "corrupt zip file",
			httpStatus: http.StatusOK,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			l := &libcnb.Layer{
				Path:     t.TempDir(),
				Metadata: map[string]interface{}{},
			}
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath(tc.responseFile)),
				testserver.WithMockURL(&dartSdkURL))

			version := "2.15.1"
			err := InstallDartSDK(ctx, l, version)

			if tc.wantError && err == nil {
				t.Fatalf("Expecting error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(l.Path, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Failed to extract. Missing file: %s (%v)", fp, err)
				}
				if l.Metadata["version"] != version {
					t.Errorf("Layer Metadata.version = %q, want %q", l.Metadata["version"], version)
				}
			}
		})
	}

}

func TestInstallRuby(t *testing.T) {
	testCases := []struct {
		name         string
		version      string
		httpStatus   int
		stackID      string
		responseFile string
		wantFile     string
		wantVersion  string
		wantError    bool
		wantCached   bool
	}{
		{
			name:         "successful install",
			version:      "2.x.x",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "2.2.2",
		},
		{
			name:         "successful cached install",
			version:      "2.2.2",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantCached:   true,
		},
		{
			name:         "default to highest available verions",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "3.3.3",
		},
		{
			name:         "invalid version",
			version:      ">9.9.9",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantError:    true,
		},
		{
			name:       "not found",
			version:    "2.2.2",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "corrupt tar file",
			version:    "2.2.2",
			httpStatus: http.StatusOK,
			wantError:  true,
		},
		{
			name:         "successful install - invalid stackID fallback to ubuntu1804",
			version:      "2.x.x",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "2.2.2",
			stackID:      "some.invalid.stackID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub the file server
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath(tc.responseFile)),
				testserver.WithMockURL(&googleTarballURL))

			// stub the version manifest
			testserver.New(
				t,
				testserver.WithStatus(http.StatusOK),
				testserver.WithJSON(`["1.1.1","3.3.3","2.2.2"]`),
				testserver.WithMockURL(&runtimeVersionsURL),
			)

			layer := &libcnb.Layer{
				Path:     t.TempDir(),
				Metadata: map[string]interface{}{},
			}
			if tc.stackID == "" {
				tc.stackID = "google.gae.18"
			}
			ctx := gcp.NewContext(gcp.WithStackID(tc.stackID))
			if tc.wantCached {
				ctx.SetMetadata(layer, "version", "2.2.2")
			}
			isCached, err := InstallTarballIfNotCached(ctx, Ruby, tc.version, layer)
			if tc.wantCached && !isCached {
				t.Fatalf("InstallTarballIfNotCached(ctx, %q, %q) got isCached: %v, want error? %v", Ruby, tc.version, isCached, tc.wantCached)
			}
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallTarballIfNotCached(ctx, %q, %q) got error: %v, want error? %v", Ruby, tc.version, err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(layer.Path, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Failed to extract. Missing file: %s (%v)", fp, err)
				}
			}
			if tc.wantVersion != "" && layer.Metadata["version"] != tc.wantVersion {
				t.Errorf("Layer Metadata.version = %q, want %q", layer.Metadata["version"], tc.wantVersion)
			}
		})
	}
}
