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
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestResolvePackageVersion(t *testing.T) {
	testCases := []struct {
		name       string
		env        string
		httpStatus int
		response   string
		want       string
		wantError  bool
	}{
		{
			name: "from env",
			env:  "2.14.0",
			want: "2.14.0",
		},
		{
			name: "fetched version",
			response: `{
				"date": "2022-02-08",
				"version": "2.16.1",
				"revision": "0180af250ff518cc0fa494a4eb484ce11ec1e62c"
			}`,
			want: "2.16.1",
		},
		{
			name:       "bad response code",
			httpStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(tc.response),
				testserver.WithMockURL(&versionURL),
			)

			if tc.env != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.env)
			}

			got, err := DetectSDKVersion()
			if tc.wantError == (err == nil) {
				t.Errorf(`DetectSDKVersion() got error: %v, want error?: %v`, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf(`DetectSDKVersion() = %q, want %q`, got, tc.want)
			}
		})
	}
}
