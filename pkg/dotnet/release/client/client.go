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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	// msftVersionURL responds with the latest version of .NET for a given release channel. The .NET
	// 6.0 SDK broke our .NET buildpack so temporarily pin the version to 3.1.x.
	msftVersionURL = "https://dotnetcli.blob.core.windows.net/dotnet/Sdk/3.1/latest.version"
	// msftReleasesIndexURL responds with an index for each release 'channel'
	msftReleasesIndexURL = "https://dotnetcli.blob.core.windows.net/dotnet/release-metadata/releases-index.json"
)

type releasesIndexResult struct {
	ReleasesIndex []ReleaseIndex `json:"releases-index"`
}

// ReleaseIndex contains index information pertaining to a channel of releases.
type ReleaseIndex struct {
	ChannelVersion    string `json:"channel-version"`
	LatestRelease     string `json:"latest-release"`
	LatestReleaseDate string `json:"latest-release-date"`
	Security          bool   `json:"security"`
	LatestRuntime     string `json:"latest-runtime"`
	LatestSDK         string `json:"latest-sdk"`
	Product           string `json:"product"`
	SupportPhase      string `json:"support-phase"`
	EOLDate           string `json:"eol-date"`
	ReleasesJSON      string `json:"releases.json"`
}

// ReleasesJSON contains a list of releases for a channel of releases
type ReleasesJSON struct {
	ChannelVersion    string    `json:"channel-version"`
	LatestRelease     string    `json:"latest-release"`
	LatestReleaseDate string    `json:"latest-release-date"`
	LatestRuntime     string    `json:"latest-runtime"`
	LatestSDK         string    `json:"latest-sdk"`
	SupportPhase      string    `json:"support-phase"`
	EOLDate           string    `json:"eol-date"`
	LifecyclePolicy   string    `json:"lifecycle-policy"`
	Releases          []Release `json:"releases"`
}

// Release is part of ReleasesJSON.
type Release struct {
	ReleaseDate       string            `json:"release-date"`
	ReleaseVersion    string            `json:"release-version"`
	Security          bool              `json:"security"`
	CVEList           []interface{}     `json:"cve-list,omitempty"`
	ReleaseNotes      string            `json:"release-notes"`
	Runtime           Runtime           `json:"runtime"`
	SDK               SDK               `json:"sdk"`
	SDKs              []SDK             `json:"sdks"`
	ASPNETCoreRuntime ASPNETCoreRuntime `json:"aspnetcore-runtime"`
	WindowsDesktop    WindowsDesktop    `json:"windowsdesktop"`
}

// Runtime is part of ReleasesJSON.
type Runtime struct {
	Version        string `json:"version"`
	VersionDisplay string `json:"version-display"`
	VsVersion      string `json:"vs-version"`
	VsMacVersion   string `json:"vs-mac-version"`
	Files          []struct {
		Name string `json:"name"`
		Rid  string `json:"rid"`
		URL  string `json:"url"`
		Hash string `json:"hash"`
	} `json:"files"`
}

// SDK is part of ReleasesJSON.
type SDK struct {
	Version        string `json:"version"`
	VersionDisplay string `json:"version-display"`
	RuntimeVersion string `json:"runtime-version"`
	VsVersion      string `json:"vs-version"`
	VsMacVersion   string `json:"vs-mac-version"`
	VsSupport      string `json:"vs-support"`
	VsMacSupport   string `json:"vs-mac-support"`
	CsharpVersion  string `json:"csharp-version"`
	FsharpVersion  string `json:"fsharp-version"`
	VbVersion      string `json:"vb-version"`
	Files          []struct {
		Name string `json:"name"`
		Rid  string `json:"rid"`
		URL  string `json:"url"`
		Hash string `json:"hash"`
	} `json:"files"`
}

// ASPNETCoreRuntime is part of ReleasesJSON.
type ASPNETCoreRuntime struct {
	Version                 string   `json:"version"`
	VersionDisplay          string   `json:"version-display"`
	VersionAspnetcoremodule []string `json:"version-aspnetcoremodule"`
	VsVersion               string   `json:"vs-version"`
	Files                   []struct {
		Name  string `json:"name"`
		Rid   string `json:"rid"`
		URL   string `json:"url"`
		Hash  string `json:"hash"`
		Akams string `json:"akams,omitempty"`
	} `json:"files"`
}

// WindowsDesktop is part of ReleasesJSON.
type WindowsDesktop struct {
	Version        string `json:"version"`
	VersionDisplay string `json:"version-display"`
	Files          []struct {
		Name string `json:"name"`
		Rid  string `json:"rid"`
		URL  string `json:"url"`
		Hash string `json:"hash"`
	} `json:"files"`
}

// Client is the release client.
type Client struct {
	versionURL       string
	releasesIndexURL string
}

// New returns a new release client which will use the official msft endpoints.
func New() *Client {
	return NewWithEndpoints(msftVersionURL, msftReleasesIndexURL)
}

// NewWithEndpoints returns a new release client which will use the given endpoints, this is useful for unit testing.
func NewWithEndpoints(versionURL, releasesIndexURL string) *Client {
	return &Client{
		versionURL:       versionURL,
		releasesIndexURL: releasesIndexURL,
	}
}

// GetReleasesIndex returns a list of release indexes where each item corresponds to a release channel.
func (rc *Client) GetReleasesIndex() ([]ReleaseIndex, error) {
	var resp releasesIndexResult
	if err := getUnmarshalledResponse(rc.releasesIndexURL, &resp); err != nil {
		return nil, err
	}
	return resp.ReleasesIndex, nil
}

// GetReleasesJSON returns the ReleasesJSON for a given ReleaseIndex.
func (rc *Client) GetReleasesJSON(releaseIndex ReleaseIndex) (*ReleasesJSON, error) {
	var resp ReleasesJSON
	if err := getUnmarshalledResponse(releaseIndex.ReleasesJSON, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLatestSDKVersion returns most recent .NET SDK version.
func (rc *Client) GetLatestSDKVersion() (string, error) {
	bytes, err := getResponse(rc.versionURL)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func getUnmarshalledResponse(url string, v interface{}) error {
	bytes, err := getResponse(url)
	if err != nil {
		return fmt.Errorf("getting url %q: %w", url, err)
	}
	if err := json.Unmarshal(bytes, v); err != nil {
		return fmt.Errorf("unmarshalling response from %q: %w", url, err)
	}
	return nil
}

func getResponse(url string) ([]byte, error) {
	client := newRetryableHTTPClient()
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getting %q: %w", url, err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		msg := fmt.Sprintf("unexpected status code: %d (%s)", response.StatusCode, http.StatusText(response.StatusCode))
		if len(bytes) > 0 {
			msg = fmt.Sprintf("%v: %v", msg, string(bytes))
		}
		return nil, fmt.Errorf(msg)
	}
	return bytes, nil
}

func newRetryableHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	return retryClient.StandardClient()
}
