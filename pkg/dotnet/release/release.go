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

// Package release contains utilities for working with SDK release versions.
package release

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
)

var (
	// ErrNoMatch is returned when GetRuntimeVersionForSDKVersion is not able to find the associated
	// .NET Runtime Version for a given .NET SDK version.
	ErrNoMatch = fmt.Errorf("no match")
)

// GetRuntimeVersionForSDKVersion returns the associated runtime version for a given SDK version.
// A given .NET SDK has an bundled version of the .NET runtime. This function returns that version
// by querying the .NET release index.
func GetRuntimeVersionForSDKVersion(releaseClient *client.Client, sdkVersion string) (string, error) {
	releasesIndex, err := releaseClient.GetReleasesIndex()
	if err != nil {
		return "", fmt.Errorf("getting releases index: %w", err)
	}
	ri, err := findReleaseIndexForSDKVersion(releasesIndex, sdkVersion)
	if err != nil {
		return "", fmt.Errorf("finding release index match for %q: %w", sdkVersion, err)
	}
	releasesJSON, err := releaseClient.GetReleasesJSON(*ri)
	if err != nil {
		return "", fmt.Errorf("getting releases json for channel %q: %w", ri.ChannelVersion, err)
	}
	sdk, err := findSDKForSDKVersion(releasesJSON, sdkVersion)
	if err != nil {
		return "", fmt.Errorf("finding release for sdk %q in channel %q: %w", sdkVersion, ri.ChannelVersion, err)
	}
	return sdk.RuntimeVersion, nil
}

func findReleaseIndexForSDKVersion(releasesIndex []client.ReleaseIndex, sdkVersion string) (*client.ReleaseIndex, error) {
	for _, ri := range releasesIndex {
		if strings.HasPrefix(sdkVersion, ri.ChannelVersion) {
			return &ri, nil
		}
	}
	return nil, ErrNoMatch
}

func findSDKForSDKVersion(releasesJSON *client.ReleasesJSON, sdkVersion string) (*client.SDK, error) {
	for _, r := range releasesJSON.Releases {
		for _, s := range r.SDKs {
			if s.Version == sdkVersion {
				return &s, nil
			}
		}
	}
	return nil, ErrNoMatch
}
