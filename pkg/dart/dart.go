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

// Package dart provides utility methods for building Dart applications.
package dart

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/hashicorp/go-retryablehttp"
)

var versionURL = "https://storage.googleapis.com/dart-archive/channels/stable/release/latest/VERSION"

// releaseInfo contains information about a Dart SDK release.
type releaseInfo struct {
	Date     string `json:"date"`
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

// DetectSDKVersion detects which SDK version should be installed from the environment or fetches
// the latest stable available version.
func DetectSDKVersion() (string, error) {
	if envVersion := os.Getenv(env.RuntimeVersion); envVersion != "" {
		return envVersion, nil
	}
	return fetchLatestSdkVersion()
}

func fetchLatestSdkVersion() (string, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3

	resp, err := retryClient.StandardClient().Get(versionURL)
	if err != nil {
		return "", buildererror.InternalErrorf("fetching Dart SDK version from %q: %v", versionURL, err)
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", buildererror.InternalErrorf("reading response: %v", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", buildererror.InternalErrorf("unexpected status code from %q: %d (%s)", versionURL, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var info releaseInfo
	if err := json.Unmarshal(bytes, &info); err != nil {
		return "", buildererror.InternalErrorf("unmarshalling response from %q: %v", versionURL, err)
	}
	return info.Version, nil
}
