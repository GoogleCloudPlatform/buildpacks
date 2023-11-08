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
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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
			layer.Cache = true
			if tc.stackID == "" {
				tc.stackID = "google.gae.18"
			}
			ctx := gcp.NewContext(gcp.WithStackID(tc.stackID))
			if tc.wantCached {
				ctx.SetMetadata(layer, versionKey, "2.2.2")
				ctx.SetMetadata(layer, stackKey, tc.stackID)
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

func TestInstallSource(t *testing.T) {
	testCases := []struct {
		name               string
		runtime            InstallableRuntime
		version            string
		httpStatus         int
		stackID            string
		responseFile       string
		runtimeImageRegion string
		wantFile           string
		wantVersion        string
		wantError          bool
	}{
		{
			name:         "install with lorry",
			runtime:      Ruby,
			version:      "2.x.x",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "2.2.2",
		},
		{
			name:               "install with artifact registry",
			runtime:            Python,
			version:            "2.x.x",
			runtimeImageRegion: "us-west1",
			wantError:          true,
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
				Metadata: map[string]any{},
			}

			layer.Cache = true
			if tc.stackID == "" {
				tc.stackID = "google.gae.18"
			}
			ctx := gcp.NewContext(gcp.WithStackID(tc.stackID))
			if tc.runtimeImageRegion != "" {
				t.Setenv(env.RuntimeImageRegion, tc.runtimeImageRegion)
			}
			_, err := InstallTarballIfNotCached(ctx, tc.runtime, tc.version, layer)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallTarballIfNotCached(ctx, %q, %q) got error: %v, want error? %v", Python, tc.version, err, tc.wantError)
			}

			if !tc.wantError {
				if tc.wantFile != "" {
					fp := filepath.Join(layer.Path, tc.wantFile)
					if _, err := os.Stat(fp); err != nil {
						t.Errorf("Failed to extract. Missing file: %s (%v)", fp, err)
					}
				}
				if tc.wantVersion != "" && layer.Metadata["version"] != tc.wantVersion {
					t.Errorf("Layer Metadata.version = %q, want %q", layer.Metadata["version"], tc.wantVersion)
				}
			}
		})
	}
}

func TestPinGemAndBundlerVersion(t *testing.T) {
	testCases := []struct {
		name         string
		version      string
		wantRubygems string
		wantBundler1 string
		wantBundler2 string
		fail         bool
		wantError    string
		mocks        []*mockprocess.Mock
	}{
		{
			name:         "Ruby 3.0.x uses rubygems 3.2.26",
			version:      "3.0.x",
			wantRubygems: "3.2.26",
			wantBundler1: "1.17.3",
			wantBundler2: "2.1.4",
		},
		{
			name:         "Ruby 2.x uses rubygems 3.1.2",
			version:      "2.x.x",
			wantRubygems: "3.1.2",
			wantBundler1: "1.17.3",
			wantBundler2: "2.1.4",
		},
		{
			name:         "Ruby 3.2+ uses rubygems 3.3.15",
			version:      "3.2.x",
			wantRubygems: "3.3.15",
			wantBundler2: "2.1.4",
		},
		{
			name:    "gem update fails",
			version: "2.7.6",
			mocks: []*mockprocess.Mock{mockprocess.New(".*gem update.*", mockprocess.WithExitCode(1),
				mockprocess.WithStderr("internal error reason"))},
			fail:      true,
			wantError: "internal error reason",
		},
		{
			name:    "gem install bundle fails",
			version: "2.7.6",
			mocks: []*mockprocess.Mock{mockprocess.New(".*gem install.*", mockprocess.WithExitCode(1),
				mockprocess.WithStderr("Bundle update failure reason"))},
			fail:      true,
			wantError: "Bundle update failure reason",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			logger := log.New(buf, "", 0)

			opts := []gcp.ContextOption{gcp.WithLogger(logger)}

			eCmd, err := mockprocess.NewExecCmd(tc.mocks...)
			if err != nil {
				t.Fatalf("PinGemAndBundlerVersion(ctx, %q, l) - error creating mock exec command: %v",
					tc.version, err)
			}
			opts = append(opts, gcp.WithExecCmd(eCmd))

			ctx := gcpbuildpack.NewContext(opts...)

			layer := &libcnb.Layer{
				Path:     t.TempDir(),
				Metadata: map[string]any{},
			}

			err = PinGemAndBundlerVersion(ctx, tc.version, layer)
			if err != nil && !tc.fail {
				t.Errorf("TesPinGemAndBundlerVersion(ctx, %q, l) got error \n%q\n want nil", tc.version, err)
			}
			if err == nil && tc.fail {
				t.Errorf("TestPinGemAndBundlerVersion(ctx, %q, l) got error \nnil\n want %q",
					tc.version, tc.wantError)
			}

			logOutput := buf.String()

			if tc.wantRubygems != "" {
				wantRubygemsLog := fmt.Sprintf("Installing RubyGems %s", tc.wantRubygems)

				if !strings.Contains(logOutput, wantRubygemsLog) {
					t.Errorf(
						"PinGemAndBundlerVersion(ctx, %q, l) log output does not contain expected rubygems string: %s",
						tc.version, wantRubygemsLog)
				}
			}

			if tc.wantBundler2 != "" {
				wantBundlerLog := fmt.Sprintf("Installing bundler %s", tc.wantBundler2)
				if tc.wantBundler1 != "" {
					wantBundlerLog = fmt.Sprintf("Installing bundler %s and %s", tc.wantBundler1, tc.wantBundler2)
				}
				if !strings.Contains(logOutput, wantBundlerLog) {
					t.Errorf(
						"PinGemAndBundlerVersion(ctx, %q, l) log output does not contain expected bundler string: %s",
						tc.version, wantBundlerLog)
				}
			}

			if tc.wantError != "" {
				if !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("PinGemAndBundlerVersion(ctx, %q, l) error = %s, want %s", tc.version,
						err.Error(), tc.wantError)
				}
			}
		})
	}
}

func TestRuntimeImageURL(t *testing.T) {
	testCases := []struct {
		runtime InstallableRuntime
		osName  string
		version string
		region  string
		want    string
	}{
		{
			runtime: "python",
			osName:  "ubuntu1804",
			version: "3.7.2",
			region:  "us",
			want:    "us-docker.pkg.dev/gae-runtimes/runtimes-ubuntu1804/python:3.7.2",
		},
		{
			runtime: "nodejs",
			osName:  "ubuntu2204",
			version: "18.18.1",
			region:  "eu",
			want:    "eu-docker.pkg.dev/gae-runtimes/runtimes-ubuntu2204/nodejs:18.18.1",
		},
		{
			runtime: "php",
			osName:  "ubuntu2204",
			version: "8.2.0",
			region:  "us-west1",
			want:    "us-west1-docker.pkg.dev/gae-runtimes/runtimes-ubuntu2204/php:8.2.0",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s-%s-%s", tc.runtime, tc.osName, tc.version, tc.region), func(t *testing.T) {
			runtimeImageURL := runtimeImageURL(tc.runtime, tc.osName, tc.version, tc.region)

			if runtimeImageURL != tc.want {
				t.Errorf("runtimeImageURL got %s, want %s", runtimeImageURL, tc.want)
			}
		})
	}
}
