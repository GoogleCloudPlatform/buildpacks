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

package dart

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/hashicorp/go-retryablehttp"
	"gopkg.in/yaml.v2"
)

var flutterVersionURL = "https://storage.googleapis.com/flutter_infra_release/releases/releases_linux.json"

// releaseDetail contains information about specific releases
type releaseDetail struct {
	Hash           string    `json:"hash"`
	Channel        string    `json:"channel"`
	Version        string    `json:"version"`
	DartSdkVersion string    `json:"dart_sdk_version,omitempty"`
	DartSdkArch    string    `json:"dart_sdk_arch,omitempty"`
	ReleaseDate    time.Time `json:"release_date"`
	Archive        string    `json:"archive"`
	Sha256         string    `json:"sha256"`
}

// flutterReleaseInfo contains information about a the current releases
type flutterReleaseInfo struct {
	BaseURL        string `json:"base_url"`
	CurrentRelease struct {
		Beta   string `json:"beta"`
		Dev    string `json:"dev"`
		Stable string `json:"stable"`
	} `json:"current_release"`
	Releases []releaseDetail `json:"releases"`
}

// Buildpack contains configuration for the flutter buildpack.
type Buildpack struct {
	Server    *string `yaml:"server"`
	Static    *string `yaml:"static"`
	Prebuild  *string `yaml:"prebuild"`
	Postbuild *string `yaml:"postbuild"`
}

// Pubspec represents a small view of a pubspec.yaml.
type Pubspec struct {
	Dependencies    map[string]any `yaml:"dependencies"`
	DevDependencies map[string]any `yaml:"dev_dependencies"`
	Buildpack       *Buildpack     `yaml:"buildpack"`
}

// findStableRelease searches for the stable release based on CurrentRelease.Stable hash.
func findStableRelease(info flutterReleaseInfo) (releaseDetail, bool) {
	stableHash := info.CurrentRelease.Stable
	if stableHash == "" {
		return releaseDetail{}, false
	}

	for _, release := range info.Releases {
		if release.Hash == stableHash {
			return release, true
		}
	}
	return releaseDetail{}, false
}

func findSpecificRelease(version string, info flutterReleaseInfo) (releaseDetail, bool) {
	for _, release := range info.Releases {
		if release.Version == version {
			return release, true
		}
	}
	return releaseDetail{}, false
}

// DetectFlutterSDKArchive detects which SDK version should be installed from the environment or fetches
// the latest stable available version. Returns version, url
func DetectFlutterSDKArchive() (string, string, error) {
	if envVersion := os.Getenv(env.RuntimeVersion); envVersion != "" {
		detail, err := fetchSpecificSdkArchive(envVersion)
		if err != nil {
			return "", "", err
		}
		return detail.Version, detail.Archive, nil
	}
	detail, err := fetchLatestSdkArchive()
	if err != nil {
		return "", "", err
	}
	return detail.Version, detail.Archive, nil
}

func downloadManifest() (flutterReleaseInfo, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3

	resp, err := retryClient.StandardClient().Get(flutterVersionURL)
	if err != nil {
		return flutterReleaseInfo{}, buildererror.InternalErrorf("fetching Dart SDK version from %q: %v", flutterVersionURL, err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return flutterReleaseInfo{}, buildererror.InternalErrorf("reading response: %v", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return flutterReleaseInfo{}, buildererror.InternalErrorf("unexpected status code from %q: %d (%s)", flutterVersionURL, resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var info flutterReleaseInfo
	if err := json.Unmarshal(bytes, &info); err != nil {
		return flutterReleaseInfo{}, buildererror.InternalErrorf("unmarshalling response from %q: %v", flutterVersionURL, err)
	}
	return info, nil
}

func fetchSpecificSdkArchive(version string) (releaseDetail, error) {
	info, err := downloadManifest()
	if err != nil {
		return releaseDetail{}, err
	}

	detail, found := findSpecificRelease(version, info)
	if !found {
		return releaseDetail{}, buildererror.InternalErrorf("version not found %q", version)
	}
	return detail, nil
}

func fetchLatestSdkArchive() (releaseDetail, error) {
	info, err := downloadManifest()
	if err != nil {
		return releaseDetail{}, err
	}

	detail, found := findStableRelease(info)
	if !found {
		return releaseDetail{}, buildererror.InternalErrorf("stablke version not found")
	}
	return detail, nil
}

// IsFlutter returns true if the given Dart project contains a pubspec.yaml that declares a
// dependency on flutter.
func IsFlutter(dir string) (bool, error) {
	ps, err := GetPubspec(dir)
	if err != nil {
		return false, gcp.UserErrorf("unmarshalling pubspec.yaml: %v", err)
	}

	if _, exists := ps.Dependencies["flutter"]; exists {
		return true, nil
	}
	return false, nil
}

// GetPubspec unmarshals the pubspec.yaml file into a Pubspec struct.
func GetPubspec(dir string) (Pubspec, error) {
	f := filepath.Join(dir, "pubspec.yaml")
	rawpjs, err := ioutil.ReadFile(f)
	if os.IsNotExist(err) {
		// If there is no pubspec.yaml, there is no build_runner dependency.
		return Pubspec{}, nil
	}
	if err != nil {
		return Pubspec{}, gcp.InternalErrorf("reading pubspec.yaml: %v", err)
	}

	var ps Pubspec
	if err := yaml.Unmarshal(rawpjs, &ps); err != nil {
		return Pubspec{}, gcp.UserErrorf("unmarshalling pubspec.yaml: %v", err)
	}

	if ps.Buildpack != nil {
		// Only insure defaults if the buildpack key is defined.
		if ps.Buildpack.Server == nil {
			server := "server"
			ps.Buildpack.Server = &server
		}
		if ps.Buildpack.Static == nil {
			static := "static"
			ps.Buildpack.Static = &static
		}
	}
	return ps, nil
}
