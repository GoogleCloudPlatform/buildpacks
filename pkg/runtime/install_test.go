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
	"net/http/httptest"
	"path/filepath"
	"testing"

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

			stubFileServer(t, tc.httpStatus, tc.responseFile)

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
				if !ctx.FileExists(fp) {
					t.Errorf("Failed to extract. Missing file: %s", fp)
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
		httpStatus   int
		responseFile string
		wantFile     string
		wantError    bool
	}{
		{
			name:         "successful install",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
		},
		{
			name:       "invalid version",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "corrupt tar file",
			httpStatus: http.StatusOK,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stubFileServer(t, tc.httpStatus, tc.responseFile)

			layer := &libcnb.Layer{
				Path:     t.TempDir(),
				Metadata: map[string]interface{}{},
			}
			ctx := gcp.NewContext()

			version := "3.0.3"
			err := InstallTarball(ctx, Ruby, version, layer)

			if tc.wantError == (err == nil) {
				t.Fatalf("InstallTarball(ctx, %q, %q) got error: %v, want error? %v", Ruby, version, err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(layer.Path, tc.wantFile)
				if !ctx.FileExists(fp) {
					t.Errorf("Failed to extract. Missing file: %s", fp)
				}
				if layer.Metadata["version"] != version {
					t.Errorf("Layer Metadata.version = %q, want %q", layer.Metadata["version"], version)
				}
			}
		})
	}
}

func stubFileServer(t *testing.T, httpStatus int, responseFile string) {
	t.Helper()
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpStatus != 0 {
			w.WriteHeader(httpStatus)
		}
		if r.UserAgent() != gcpUserAgent {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if responseFile != "" {
			http.ServeFile(w, r, testdata.MustGetPath(responseFile))
		}
	}))
	t.Cleanup(svr.Close)

	origDartURL := dartSdkURL
	origTarballURL := googleTarballURL
	t.Cleanup(func() {
		dartSdkURL = origDartURL
		googleTarballURL = origTarballURL
	})
	dartSdkURL = svr.URL + "?version=%s"
	googleTarballURL = svr.URL + "?runtime=%s&version=%s"
}
