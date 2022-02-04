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

package nodejs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestPackageVersion(t *testing.T) {
	testCases := []struct {
		name       string
		httpStatus int
		response   string
		want       string
		wantError  bool
	}{
		{
			name: "with next version",
			response: `{
				"name": "npm",
				"dist-tags": {
					"latest": "8.4.0",
					"next": "8.0.0-rc.1"
				},
				"versions": {
					"8.0.0-rc.1": {
						"name": "npm",
						"version": "8.0.0-rc.1"
					},
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "8.4.0",
		},
		{
			name:       "non-existent package",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stubNPMRegistry(t, tc.response, tc.httpStatus)

			got, err := latestPackageVersion("npm")

			if tc.wantError == (err == nil) {
				t.Fatalf(`latestPackageVersion("npm") got error: %v, want error? %v`, err, tc.wantError)
			}
			if got != tc.want {
				t.Fatalf(`latestPackageVersion("npm") = %q, want %q`, got, tc.want)
			}
		})
	}
}

func TestResolvePackageVersion(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		httpStatus int
		response   string
		want       string
		wantError  bool
	}{
		{
			name:       "single version",
			constraint: "8.x.x",
			response: `{
				"name": "npm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "8.4.0",
		},
		{
			name:       "multiple versions",
			constraint: "<9.0.0",
			response: `{
				"name": "npm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					},
					"8.5.0": {
						"name": "npm",
						"version": "8.5.0"
					},
					"9.0.0": {
						"name": "npm",
						"version": "9.0.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "8.5.0",
		},
		{
			name:       "non-existent package",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "invalid constraint",
			constraint: "invalid",
			response: `{
				"name": "npm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stubNPMRegistry(t, tc.response, tc.httpStatus)

			got, err := resolvePackageVersion("npm", tc.constraint)
			if tc.wantError == (err == nil) {
				t.Fatalf(`resolvePackageVersion("npm", %q) got error: %v, want error?: %v`, tc.constraint, err, tc.wantError)
			}
			if got != tc.want {
				t.Fatalf(`resolvePackageVersion("npm", %q) = %q, want %q`, tc.constraint, got, tc.want)
			}
		})
	}
}

func stubNPMRegistry(t *testing.T, responseData string, httpStatus int) {
	t.Helper()

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if httpStatus != 0 {
			w.WriteHeader(httpStatus)
		}
		w.Write([]byte(responseData))
	}))
	t.Cleanup(svr.Close)

	origURL := npmRegistryURL
	t.Cleanup(func() { npmRegistryURL = origURL })
	npmRegistryURL = svr.URL + "?package=%s"
}
