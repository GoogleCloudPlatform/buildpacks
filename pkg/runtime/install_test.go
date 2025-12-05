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
	"errors"
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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/tpc"
	"github.com/buildpacks/libcnb/v2"
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
				Metadata: map[string]any{},
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

func TestInstallFlutterSDK(t *testing.T) {
	testCases := []struct {
		name         string
		httpStatus   int
		responseFile string
		wantFile     string
		wantError    bool
	}{
		{
			name:         "successful install",
			responseFile: "testdata/dummy-flutter-sdk.tar.xz",
			wantFile:     "bin/flutter",
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
			ctx := gcp.NewContext()
			l := &libcnb.Layer{
				Path:     t.TempDir(),
				Metadata: map[string]any{},
			}
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath(tc.responseFile)),
				testserver.WithMockURL(&flutterSdkURL))

			version := "3.29.3"
			err := InstallFlutterSDK(ctx, l, version, "stable/linux/flutter_linux_3.29.3-stable.tar.xz")

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
			name:         "successful_install",
			version:      "2.x.x",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "2.2.2",
		},
		{
			name:         "successful_cached_install",
			version:      "2.2.2",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantCached:   true,
		},
		{
			name:         "default_to_highest_available_verions",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "3.3.3",
		},
		{
			name:         "invalid_version",
			version:      ">9.9.9",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantError:    true,
		},
		{
			name:       "not_found",
			version:    "2.2.2",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "corrupt_tar_file",
			version:    "2.2.2",
			httpStatus: http.StatusOK,
			wantError:  true,
		},
		{
			name:         "successful_install-invalid_stackID_fallback_to_ubuntu1804",
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
				Metadata: map[string]any{},
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
		name                       string
		runtime                    InstallableRuntime
		version                    string
		httpStatus                 int
		stackID                    string
		responseFile               string
		runtimeImageRegion         string
		buildEnv                   string
		buildUniverse              string
		wantFile                   string
		wantVersion                string
		wantError                  bool
		wantAR                     bool
		serverlessRuntimesTarballs string
	}{
		{
			name:         "Lorry_when_region_is_not_set",
			runtime:      Ruby,
			version:      "2.x.x",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantFile:     "lib/foo.txt",
			wantVersion:  "2.2.2",
			wantAR:       false,
		},
		{
			name:               "Lorry_when_GOOGLE_BUILD_ENV_is_dev",
			runtime:            Ruby,
			version:            "2.x.x",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			buildEnv:           "dev",
			runtimeImageRegion: "us-west1",
			wantFile:           "lib/foo.txt",
			wantVersion:        "2.2.2",
			wantAR:             false,
		},
		{
			name:               "Lorry_for_Go_runtime_even_when_region_and_prod_env_are_set",
			runtime:            Go,
			version:            "1.24.5",
			runtimeImageRegion: "us-west1",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			buildEnv:           "prod",
			wantAR:             false,
		},
		{
			name:               "AR_QA_when_GOOGLE_BUILD_ENV_is_qual",
			runtime:            Python,
			version:            "3.10.0",
			runtimeImageRegion: "us-west1",
			buildEnv:           "qual",
			wantAR:             true,
		},
		{
			name:               "AR_Prod_when_GOOGLE_BUILD_ENV_is_prod",
			runtime:            Nodejs,
			version:            "16.20.0",
			runtimeImageRegion: "us-central1",
			buildEnv:           "prod",
			wantAR:             true,
		},
		{
			name:               "AR_Prod_when_GOOGLE_BUILD_ENV_is_not_set",
			runtime:            Nodejs,
			version:            "16.20.0",
			runtimeImageRegion: "us-central1",
			wantAR:             true,
		},
		{
			name:               "install_with_artifact_registry_1",
			runtime:            Python,
			version:            "3.10.0",
			runtimeImageRegion: "us-west1",
			wantError:          false,
			wantAR:             true,
		},
		{
			name:                       "install_with_artifact_registry_serverless_runtimes",
			runtime:                    Python,
			version:                    "3.10.0",
			runtimeImageRegion:         "us-west1",
			wantError:                  false,
			wantAR:                     true,
			serverlessRuntimesTarballs: "true",
		},
		{
			name:               "install_from_artifact_registry_2",
			runtime:            Nodejs,
			version:            "16.20.0",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			runtimeImageRegion: "us-west1",
			wantError:          false,
			wantAR:             true,
		},
		{
			name:               "install_from_artifact_registry_3",
			runtime:            OpenJDK,
			version:            "17.1.0",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			runtimeImageRegion: "us-central1",
			wantError:          false,
			wantAR:             true,
		},
		{
			name:               "install_from_artifact_registry_4",
			runtime:            Jetty,
			version:            "latest",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			runtimeImageRegion: "us-central1",
			wantError:          false,
			wantAR:             true,
		},
		{
			name:               "install_from_artifact_registry_for_java_21.0",
			runtime:            CanonicalJDK,
			version:            "21.0",
			stackID:            "google.gae.22",
			responseFile:       "testdata/dummy-ruby-runtime.tar.gz",
			runtimeImageRegion: "us-central1",
			wantError:          false,
			wantAR:             true,
		},
		{
			name:         "missing_runtimeImageRegion",
			runtime:      Nodejs,
			version:      "16.20.0",
			responseFile: "testdata/dummy-ruby-runtime.tar.gz",
			wantError:    false,
			wantAR:       false,
		},
		{
			name:               "AR_Prod_for_jdk_version_with_beta_and_stable_present",
			runtime:            OpenJDK,
			version:            "25.0.1-beta",
			runtimeImageRegion: "us-central1",
			buildEnv:           "prod",
			wantAR:             true,
			wantVersion:        "25.0.1-beta",
		},
		{
			name:               "AR_Prod_for_jdk_version_with_beta_exact",
			runtime:            OpenJDK,
			version:            "25.0.1_12-beta",
			runtimeImageRegion: "us-central1",
			buildEnv:           "prod",
			wantAR:             true,
			wantVersion:        "25.0.1_12-beta",
		},
		{
			name:               "AR_TPC_prp",
			runtime:            Nodejs,
			version:            "16.20.0",
			runtimeImageRegion: "u-us-prp1",
			buildUniverse:      "prp",
			wantAR:             true,
		},
		{
			name:               "AR_TPC_tsp",
			runtime:            Nodejs,
			version:            "16.20.0",
			runtimeImageRegion: "u-germany-northeast1",
			buildUniverse:      "tsp",
			wantAR:             true,
		},
		{
			name:               "AR_TPC_tsq",
			runtime:            Nodejs,
			version:            "16.20.0",
			runtimeImageRegion: "u-germany-northeast1q",
			buildUniverse:      "tsq",
			wantAR:             true,
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

			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath(tc.responseFile)),
				testserver.WithMockURL(&goTarballURL))

			// stub the version manifest
			testserver.New(
				t,
				testserver.WithStatus(http.StatusOK),
				testserver.WithJSON(`["1.1.1","3.3.3","2.2.2","16.20.0"]`),
				testserver.WithMockURL(&runtimeVersionsURL),
			)
			fetchedFromAR := false
			defer func(fn func(url, fallbackURL, dir string, stripComponents int, ctx *gcp.Context) error) {
				fetch.ARImage = fn
			}(fetch.ARImage)

			fetch.ARImage = func(url, fallbackURL, dir string, stripComponents int, ctx *gcp.Context) error {
				fetchedFromAR = true
				if tc.wantAR && tc.buildUniverse != "" && tc.buildUniverse != "gdu" {
					hostname, present := tpc.ARRegionToHostname(tc.runtimeImageRegion)
					if !present {
						t.Fatalf("For TPC, invalid region %q specified", tc.runtimeImageRegion)
					}
					if !strings.Contains(url, hostname) {
						t.Errorf("For TPC, fetch.ARImage URL %q does not contain expected hostname %q", url, hostname)
					}
					project, present := tpc.UniverseToProject(tc.buildUniverse)
					if !present {
						t.Fatalf("For TPC, invalid universe %q specified", tc.buildUniverse)
					}
					if !strings.Contains(url, project) {
						t.Errorf("For TPC, fetch.ARImage URL %q does not contain expected project %q", url, project)
					}
				}
				return nil
			}

			defer func(fn func(url, fallbackURL string, ctx *gcp.Context) ([]string, error)) {
				fetch.ARVersions = fn
			}(fetch.ARVersions)

			fetch.ARVersions = func(url, fallbackURL string, ctx *gcp.Context) ([]string, error) {
				return []string{"11.0.21_9-post-Ubuntu-0ubuntu122.04", "17.0.9_9-Ubuntu-122.04", "21.0.1_12-Ubuntu-222.04", "17.0.1_12-beta", "17.0.1_13"}, nil
			}

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
			if tc.serverlessRuntimesTarballs != "" {
				t.Setenv(env.ServerlessRuntimesTarballs, tc.serverlessRuntimesTarballs)
			}
			if tc.buildEnv != "" {
				t.Setenv(env.BuildEnv, tc.buildEnv)
			}
			if tc.buildUniverse != "" {
				t.Setenv(env.BuildUniverse, tc.buildUniverse)
			}
			_, err := InstallTarballIfNotCached(ctx, tc.runtime, tc.version, layer)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallTarballIfNotCached(ctx, %q, %q) got error: %v, want error? %v", tc.runtime, tc.version, err, tc.wantError)
			}

			if tc.wantAR != fetchedFromAR {
				t.Errorf("Fetched runtime from AR = %v and want AR = %v", fetchedFromAR, tc.wantAR)
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
		name     string
		runtime  InstallableRuntime
		osName   string
		version  string
		region   string
		registry string
		want     string
	}{
		{
			name:     "python_qual",
			runtime:  "python",
			osName:   "ubuntu2204",
			version:  "3.7.2",
			region:   "us",
			registry: "serverless-runtimes-qa",
			want:     "us-docker.pkg.dev/serverless-runtimes-qa/runtimes-ubuntu2204/python:3.7.2",
		},
		{
			name:     "nodejs_prod_gae",
			runtime:  "nodejs",
			osName:   "ubuntu2204",
			version:  "18.18.1",
			region:   "eu",
			registry: "gae-runtimes",
			want:     "eu-docker.pkg.dev/gae-runtimes/runtimes-ubuntu2204/nodejs:18.18.1",
		},
		{
			name:     "php_prod_serverless",
			runtime:  "php",
			osName:   "ubuntu2204",
			version:  "8.2.0",
			region:   "us-west1",
			registry: "serverless-runtimes",
			want:     "us-west1-docker.pkg.dev/serverless-runtimes/runtimes-ubuntu2204/php:8.2.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			hostname, err := arHostname(ctx, tc.region)
			if err != nil {
				t.Fatalf("arHostname(%q) failed: %v", tc.region, err)
			}
			got := runtimeImageURL(hostname, tc.registry, tc.osName, tc.runtime, tc.version)
			if got != tc.want {
				t.Errorf("runtimeImageURL() got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetTarballRegistry(t *testing.T) {
	testCases := []struct {
		name                       string
		buildEnv                   string
		buildUniverse              string
		serverlessRuntimesTarballs string
		want                       string
	}{
		{
			name:          "dev_env",
			buildEnv:      "dev",
			want:          tarballRegistryDev,
			buildUniverse: "gdu",
		},
		{
			name:          "qual_env",
			buildEnv:      "qual",
			want:          tarballRegistryQual,
			buildUniverse: "",
		},
		{
			name:          "prod_env_without_serverless_runtimes_flag",
			buildEnv:      "prod",
			want:          tarballRegistryProdGae,
			buildUniverse: "gdu",
		},
		{
			name:                       "prod_env_with_serverless_runtimes_flag",
			buildEnv:                   "prod",
			serverlessRuntimesTarballs: "true",
			want:                       tarballRegistryProdServerless,
			buildUniverse:              "gdu",
		},
		{
			name:          "unspecified_env_defaults_to_prod",
			want:          tarballRegistryProdGae,
			buildUniverse: "gdu",
		},
		{
			name:                       "unspecified_env_with_serverless_runtimes_flag",
			serverlessRuntimesTarballs: "true",
			want:                       tarballRegistryProdServerless,
			buildUniverse:              "gdu",
		},
		{
			name:          "prp_universe",
			buildUniverse: "prp",
			want:          "tpczero-system/serverless-runtimes-tpc",
		},
		{
			name:          "tsp_universe",
			buildUniverse: "tsp",
			want:          "eu0-system/serverless-runtimes-tpc",
		},
		{
			name:          "tsq_universe",
			buildUniverse: "tsq",
			want:          "tpcone-system/serverless-runtimes-tpc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.buildEnv != "" {
				t.Setenv(env.BuildEnv, tc.buildEnv)
			}
			if tc.buildUniverse != "" {
				t.Setenv(env.BuildUniverse, tc.buildUniverse)
			}
			if tc.serverlessRuntimesTarballs != "" {
				t.Setenv(env.ServerlessRuntimesTarballs, tc.serverlessRuntimesTarballs)
			}
			got := tarballRegistry()
			if got != tc.want {
				t.Errorf("tarballRegistry() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateMinFlexVersion(t *testing.T) {
	testCases := []struct {
		name       string
		version    string
		minVersion string
		runtime    InstallableRuntime
		env        string
		envRuntime string
		wantErr    bool
	}{
		{
			name:       "non language runtime pid1",
			version:    "2.8",
			minVersion: "3.7.0",
			runtime:    Pid1,
			wantErr:    false,
		},
		{
			name:       "non language runtime nginx",
			version:    "2.8",
			minVersion: "3.7.0",
			runtime:    Nginx,
			wantErr:    false,
		},
		{
			name:       "valid version",
			version:    "3.7.2",
			minVersion: "3.7.0",
			wantErr:    false,
		},
		{
			name:       "version below min version",
			version:    "2.8",
			minVersion: "3.7.0",
			wantErr:    true,
		},
		{
			name:       "invalid semver in env",
			version:    "4.3.2",
			minVersion: "cde",
			wantErr:    false,
		},
		{
			name:       "invalid semver version input",
			version:    "abc",
			minVersion: "3.7.0",
			wantErr:    true,
		},
		{
			name:    "non flex environment",
			env:     env.TargetPlatformAppEngine,
			wantErr: false,
		},
		{
			name:       "no validation if runtime does not match",
			env:        env.TargetPlatformFlex,
			version:    "abc",
			minVersion: "3.7.0",
			runtime:    Python,
			envRuntime: "php",
			wantErr:    false,
		},
	}
	t.Setenv(env.XGoogleTargetPlatform, env.TargetPlatformFlex)
	t.Setenv(env.Runtime, "python")
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := Python
			if tc.runtime != "" {
				runtime = tc.runtime
			}
			if tc.envRuntime != "" {
				t.Setenv(env.Runtime, tc.envRuntime)
			}

			ctx := gcp.NewContext()
			if tc.env != "" {
				t.Setenv(env.XGoogleTargetPlatform, tc.env)
			}
			t.Setenv(env.FlexMinVersion, tc.minVersion)
			err := ValidateFlexMinVersion(ctx, runtime, tc.version)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("ValidateMinFlexVersion(%v)= %v, want error presence: %v, ", tc.version, err, tc.wantErr)
			}

			t.Setenv(env.Runtime, "python")
		})
	}
}

func TestRuntimeMatchesInstallableRuntime(t *testing.T) {
	tests := []struct {
		installableRuntime InstallableRuntime
		env                string
		want               bool
	}{
		{
			installableRuntime: OpenJDK,
			env:                "java",
			want:               true,
		},
		{
			installableRuntime: CanonicalJDK,
			env:                "java",
			want:               true,
		},
		{
			installableRuntime: AspNetCore,
			env:                "dotnet",
			want:               true,
		},
		{
			installableRuntime: DotnetSDK,
			env:                "dotnet",
			want:               true,
		},
		{
			installableRuntime: Python,
			env:                "nodejs",
			want:               false,
		},
		{
			installableRuntime: Nodejs,
			env:                "nodejs",
			want:               true,
		},
	}

	for _, tc := range tests {
		t.Setenv(env.Runtime, tc.env)
		got := runtimeMatchesInstallableRuntime(tc.installableRuntime)
		if got != tc.want {
			t.Errorf("runtimeMatchesInstallableRuntime(%v) = %v, want: %v", tc.installableRuntime, got, tc.want)
		}
	}
}

func TestResolveVersionARFallback(t *testing.T) {
	testCases := []struct {
		name               string
		runtime            InstallableRuntime
		versionConstraint  string
		runtimeImageRegion string
		primaryARVersions  []string
		primaryARError     error
		fallbackARVersions []string
		fallbackARError    error
		wantVersion        string
		wantErr            bool
	}{
		{
			name:               "Primary OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "europe-west1",
			primaryARVersions:  []string{"18.15.0", "18.16.0"},
			fallbackARVersions: []string{"18.14.0"},
			wantVersion:        "18.16.0",
		},
		{
			name:               "Primary OK, Version NOT Found, Fallback OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.17.x",
			runtimeImageRegion: "europe-west1",
			primaryARVersions:  []string{"18.15.0", "18.16.0"},
			fallbackARVersions: []string{"18.16.0", "18.17.0"},
			wantVersion:        "18.17.0",
		},
		{
			name:               "Primary OK, Version NOT Found, Fallback OK, Version NOT Found",
			runtime:            Nodejs,
			versionConstraint:  "19.x.x",
			runtimeImageRegion: "europe-west1",
			primaryARVersions:  []string{"18.15.0", "18.16.0"},
			fallbackARVersions: []string{"18.16.0", "18.17.0"},
			wantErr:            true,
		},
		{
			name:               "Primary ERR, Fallback OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.17.x",
			runtimeImageRegion: "asia-east1",
			primaryARError:     errors.New("primary region down"),
			fallbackARVersions: []string{"18.16.0", "18.17.0"},
			wantVersion:        "18.17.0",
		},
		{
			name:               "Primary ERR, Fallback OK, Version NOT Found",
			runtime:            Nodejs,
			versionConstraint:  "19.x.x",
			runtimeImageRegion: "asia-east1",
			primaryARError:     errors.New("primary region down"),
			fallbackARVersions: []string{"18.16.0", "18.17.0"},
			wantErr:            true,
		},
		{
			name:               "Primary ERR, Fallback ERR",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "australia-southeast1",
			primaryARError:     errors.New("primary region down"),
			fallbackARError:    errors.New("fallback region down"),
			wantErr:            true,
		},
		{
			name:               "Primary OK, Version NOT Found, Fallback ERR",
			runtime:            Nodejs,
			versionConstraint:  "18.17.x",
			runtimeImageRegion: "europe-west2",
			primaryARVersions:  []string{"18.15.0", "18.16.0"},
			fallbackARError:    errors.New("fallback region down"),
			wantErr:            true,
		},
		{
			name:               "OpenJDK Primary OK, Fallback has version",
			runtime:            OpenJDK,
			versionConstraint:  "11",
			runtimeImageRegion: "europe-west3",
			primaryARVersions:  []string{"8.0.302_8"},
			fallbackARVersions: []string{"11.0.21_9", "11.0.22_10"},
			wantVersion:        "11.0.22_10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.RuntimeImageRegion, tc.runtimeImageRegion)
			t.Setenv(env.BuildEnv, "prod") // Ensure AR is used

			origARVersions := fetch.ARVersions
			defer func() {
				fetch.ARVersions = origARVersions
			}()

			fetch.ARVersions = func(url, fallbackURL string, ctx *gcp.Context) ([]string, error) {
				if fallbackURL != "" {
					return nil, errors.New("fallbackURL should be empty in fetch.ARVersions calls")
				}
				if strings.Contains(url, tc.runtimeImageRegion) {
					return tc.primaryARVersions, tc.primaryARError
				}
				if strings.Contains(url, fallbackRegion) {
					return tc.fallbackARVersions, tc.fallbackARError
				}
				return nil, fmt.Errorf("unexpected URL in ARVersions mock: %s", url)
			}

			ctx := gcp.NewContext(gcp.WithStackID("google.22"))
			gotVersion, err := ResolveVersion(ctx, tc.runtime, tc.versionConstraint, "ubuntu2204")

			if tc.wantErr {
				if err == nil {
					t.Errorf("ResolveVersion(%q, %q) succeeded, want error", tc.runtime, tc.versionConstraint)
				}
			} else {
				if err != nil {
					t.Fatalf("ResolveVersion(%q, %q) failed: %v", tc.runtime, tc.versionConstraint, err)
				}
				if gotVersion != tc.wantVersion {
					t.Errorf("ResolveVersion(%q, %q) = %q, want %q", tc.runtime, tc.versionConstraint, gotVersion, tc.wantVersion)
				}
			}
		})
	}
}

func TestResolveVersionTPC(t *testing.T) {
	testCases := []struct {
		name               string
		runtime            InstallableRuntime
		versionConstraint  string
		runtimeImageRegion string
		buildUniverse      string
		arVersions         []string
		arError            error
		wantVersion        string
		wantErr            bool
	}{
		{
			name:               "TPC PRP OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "u-us-prp1",
			buildUniverse:      "prp",
			arVersions:         []string{"18.15.0", "18.16.0"},
			wantVersion:        "18.16.0",
		},
		{
			name:               "TPC TSP OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "u-germany-northeast1",
			buildUniverse:      "tsp",
			arVersions:         []string{"18.15.0", "18.16.0"},
			wantVersion:        "18.16.0",
		},
		{
			name:               "TPC TSQ OK, Version Found",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "u-germany-northeast1q",
			buildUniverse:      "tsq",
			arVersions:         []string{"18.15.0", "18.16.0"},
			wantVersion:        "18.16.0",
		},
		{
			name:               "TPC PRP OK, Version NOT Found",
			runtime:            Nodejs,
			versionConstraint:  "19.x.x",
			runtimeImageRegion: "u-us-prp1",
			buildUniverse:      "prp",
			arVersions:         []string{"18.15.0", "18.16.0"},
			wantErr:            true,
		},
		{
			name:               "TPC PRP AR Error",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "u-us-prp1",
			buildUniverse:      "prp",
			arError:            errors.New("AR error"),
			wantErr:            true,
		},
		{
			name:               "TPC invalid region",
			runtime:            Nodejs,
			versionConstraint:  "18.x.x",
			runtimeImageRegion: "invalid-region",
			buildUniverse:      "prp",
			wantErr:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.RuntimeImageRegion, tc.runtimeImageRegion)
			t.Setenv(env.BuildUniverse, tc.buildUniverse)

			origARVersions := fetch.ARVersions
			defer func() {
				fetch.ARVersions = origARVersions
			}()

			arVersionsCalled := false
			fetch.ARVersions = func(url, fallbackURL string, ctx *gcp.Context) ([]string, error) {
				arVersionsCalled = true
				if fallbackURL != "" {
					return nil, errors.New("fallbackURL should be empty in fetch.ARVersions calls for TPC")
				}
				if hostname, present := tpc.ARRegionToHostname(tc.runtimeImageRegion); present {
					if !strings.Contains(url, hostname) {
						t.Errorf("For TPC, fetch.ARVersions URL %q does not contain expected hostname %q", url, hostname)
					}
				}
				if project, present := tpc.UniverseToProject(tc.buildUniverse); present {
					if !strings.Contains(url, project) {
						t.Errorf("For TPC, fetch.ARVersions URL %q does not contain expected project %q", url, project)
					}
				}
				return tc.arVersions, tc.arError
			}

			ctx := gcp.NewContext(gcp.WithStackID("google.22"))
			gotVersion, err := ResolveVersion(ctx, tc.runtime, tc.versionConstraint, "ubuntu2204")

			if tc.wantErr {
				if err == nil {
					t.Errorf("ResolveVersion(%q, %q) succeeded, want error", tc.runtime, tc.versionConstraint)
				}
				if tc.runtimeImageRegion == "invalid-region" && arVersionsCalled {
					t.Errorf("fetch.ARVersions should not be called for invalid region")
				}
			} else {
				if err != nil {
					t.Fatalf("ResolveVersion(%q, %q) failed: %v", tc.runtime, tc.versionConstraint, err)
				}
				if gotVersion != tc.wantVersion {
					t.Errorf("ResolveVersion(%q, %q) = %q, want %q", tc.runtime, tc.versionConstraint, gotVersion, tc.wantVersion)
				}
				if !arVersionsCalled {
					t.Errorf("fetch.ARVersions should have been called")
				}
			}
		})
	}
}
