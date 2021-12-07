// Copyright 2021 Google LLC
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

// Package client contains a client for accessing .NET release information.
package client

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	// versionURL responds with the latest version of .NET for a given release channel. The .NET 6.0
	// broke our .NET buildpack so temporarily pin the version to 3.1.x.
	versionURL = "https://dotnetcli.blob.core.windows.net/dotnet/Sdk/3.1/latest.version"
)

// GetLatestSDKVersion returns most recent .NET SDK version.
func GetLatestSDKVersion() (string, error) {
	client := newRetryableHTTPClient()
	response, err := client.Get(versionURL)
	if err != nil {
		return "", fmt.Errorf("getting %q: %w", versionURL, err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		msg := fmt.Sprintf("unexpected status code: %d (%s)", response.StatusCode, http.StatusText(response.StatusCode))
		if len(bytes) > 0 {
			msg = fmt.Sprintf("%v: %v", msg, string(bytes))
		}
		return "", fmt.Errorf(msg)
	}
	return string(bytes), nil
}

func newRetryableHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	return retryClient.StandardClient()
}
