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
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestGetLatestGradleVersion(t *testing.T) {
	testCases := []struct {
		name       string
		want       string
		httpStatus int
		response   string
		wantError  bool
	}{
		{
			name: "latest",
			response: `{
				"version" : "7.4.2",
				"buildTime" : "20220331152529+0000",
				"current" : true,
				"snapshot" : false,
				"nightly" : false,
				"releaseNightly" : false,
				"activeRc" : false,
				"rcFor" : "",
				"milestoneFor" : "",
				"broken" : false,
				"downloadUrl" : "https://services.gradle.org/distributions/gradle-7.4.2-bin.zip",
				"checksumUrl" : "https://services.gradle.org/distributions/gradle-7.4.2-bin.zip.sha256",
				"wrapperChecksumUrl" : "https://services.gradle.org/distributions/gradle-7.4.2-wrapper.jar.sha256"
			}`,
			want: "7.4.2",
		},
		{
			name:       "unavailable",
			response:   `not found`,
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stubGradleVersionService(t, tc.response, tc.httpStatus)
			got, err := GetLatestGradleVersion()
			if tc.wantError == (err == nil) {
				t.Errorf(`GetLatestGradleVersion() got error: %v, want error? %v`, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf(`GetLatestGradleVersion() = %q, want %q`, got, tc.want)
			}
		})
	}
}

func stubGradleVersionService(t *testing.T, responseData string, httpStatus int) {
	t.Helper()
	testserver.New(
		t,
		testserver.WithStatus(httpStatus),
		testserver.WithJSON(responseData),
		testserver.WithMockURL(&gradleVersionURL),
	)
}
