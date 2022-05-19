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

package java

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
)

const (
	// DefaultGradleVersion is the gradle version if not provided by the user
	DefaultGradleVersion = "6.5.1"
)

var (
	gradleVersionURL = "https://services.gradle.org/versions/current"
)

// APIResponseGradleVersion is the API response from https://services.gradle.org/versions/current
type APIResponseGradleVersion struct {
	Version            string `json:"version"`
	BuildTime          string `json:"buildTime"`
	Current            bool   `json:"current"`
	Snapshot           bool   `json:"snapshot"`
	Nightly            bool   `json:"nightly"`
	ReleaseNightly     bool   `json:"releaseNightly"`
	ActiveRc           bool   `json:"activeRc"`
	RcFor              string `json:"rcFor"`
	MilestoneFor       string `json:"milestoneFor"`
	Broken             bool   `json:"broken"`
	DownloadURL        string `json:"downloadUrl"`
	ChecksumURL        string `json:"checksumUrl"`
	WrapperChecksumURL string `json:"wrapperChecksumUrl"`
}

// GetLatestGradleVersion gets the latest gradle version if available
func GetLatestGradleVersion() string {
	var result APIResponseGradleVersion
	if err := fetch.JSON(gradleVersionURL, result); err != nil {
		return DefaultGradleVersion
	}
	return result.Version
}
