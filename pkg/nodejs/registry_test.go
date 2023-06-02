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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
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
		pkg        string
		constraint string
		httpStatus int
		response   string
		want       string
		wantError  bool
	}{
		{
			name:       "single version",
			pkg:        "npm",
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
			pkg:        "npm",
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
			pkg:        "npm",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "invalid constraint",
			pkg:        "npm",
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
		{
			name:       "deprecated version",
			pkg:        "npm",
			constraint: "<9.0.0",
			response: `{
				"name": "npm",
				"dist-tags": {
					"latest": "8.2.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0",
						"deprecated": "this version is deprecated for some reason"
					},
					"8.2.0": {
						"name": "npm",
						"version": "8.2.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "8.2.0",
		},
		{
			name:       "yarn 1",
			pkg:        "yarn",
			constraint: "1.x.x",
			response: `{
				"name": "yarn",
				"dist-tags": {
					"latest": "1.21.1"
				},
				"versions": {
					"1.21.1": {
						"name": "yarn",
						"version": "1.21.1"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "1.21.1",
		},
		{
			name:       "yarn 2",
			pkg:        "yarn",
			constraint: "2.x.x",
			response: `{
				"name": "yarn",
				"dist-tags": {
					"latest": "1.21.1"
				},
				"versions": {
					"1.21.1": {
						"name": "yarn",
						"version": "1.21.1"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "2.4.3",
		},
		{
			name:       "yarn *",
			pkg:        "yarn",
			constraint: "*",
			response: `{
				"name": "yarn",
				"dist-tags": {
					"latest": "1.21.1"
				},
				"versions": {
					"1.21.1": {
						"name": "yarn",
						"version": "1.21.1"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			want: "1.21.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stubNPMRegistry(t, tc.response, tc.httpStatus)
			stubYarnTags(t)

			got, err := resolvePackageVersion(tc.pkg, tc.constraint)
			if tc.wantError == (err == nil) {
				t.Fatalf(`resolvePackageVersion(%q, %q) got error: %v, want error?: %v`, tc.pkg, tc.constraint, err, tc.wantError)
			}
			if got != tc.want {
				t.Fatalf(`resolvePackageVersion(%q, %q) = %q, want %q`, tc.pkg, tc.constraint, got, tc.want)
			}
		})
	}
}

func stubNPMRegistry(t *testing.T, responseData string, httpStatus int) {
	t.Helper()

	testserver.New(
		t,
		testserver.WithStatus(httpStatus),
		testserver.WithJSON(responseData),
		testserver.WithMockURL(&npmRegistryURL),
	)
}

func stubYarnTags(t *testing.T) {
	t.Helper()

	testserver.New(
		t,
		testserver.WithJSON(`{
			"latest": {
				"stable": "3.2.0",
				"canary": "3.2.0"
			},
			"tags": [
				"3.2.0",
				"3.0.0",
				"3.0.0-rc.2",
				"3.0.0-rc.1",
				"2.4.3",
				"2.2.1",
				"2.0.0-rc.4"
			]
		}`),
		testserver.WithMockURL(&yarnTagsURL),
	)
}
