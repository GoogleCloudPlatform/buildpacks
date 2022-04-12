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

package fetch

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

func TestTarball(t *testing.T) {
	testCases := []struct {
		name            string
		httpStatus      int
		stripComponents int
		responseFile    string
		wantFile        string
		wantError       bool
	}{
		{
			name:         "simple untar",
			responseFile: "testdata/test.tar.gz",
			wantFile:     "lib/foo.txt",
		},
		{
			name:            "strip components",
			responseFile:    "testdata/test.tar.gz",
			stripComponents: 1,
			wantFile:        "foo.txt",
		},
		{
			name:       "not found",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:         "corrupt tar file",
			responseFile: "testdata/test.json",
			httpStatus:   http.StatusOK,
			wantError:    true,
		},
		{
			name:            "strip too many components",
			responseFile:    "testdata/test.tar.gz",
			stripComponents: 2,
			wantError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithFile(testdata.MustGetPath(tc.responseFile)))

			dir := t.TempDir()
			err := Tarball(server.URL, dir, tc.stripComponents)
			if tc.wantError == (err == nil) {
				t.Fatalf("Tarball(%q, %q, %v) got error: %v, want error? %v", server.URL, dir, tc.stripComponents, err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(dir, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Failed to extract. Missing file: %s (%v)", fp, err)
				}
			}
		})
	}
}

func TestJSON(t *testing.T) {
	testCases := []struct {
		name       string
		httpStatus int
		response   string
		wantError  bool
		want       map[string]string
	}{
		{
			name:     "simple untar",
			response: `{"foo": "bar"}`,
			want:     map[string]string{"foo": "bar"},
		},
		{
			name:       "not found",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name:       "invalid json",
			response:   "foo bar",
			httpStatus: http.StatusOK,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(tc.response))

			var got map[string]string
			err := JSON(server.URL, &got)
			if tc.wantError == (err == nil) {
				t.Fatalf("JSON(%q, &got) got error: %v, want error? %v", server.URL, err, tc.wantError)
			}
			if !cmp.Equal(got, tc.want) {
				t.Errorf("JSON(%q, &got) = %v, want %v", server.URL, got, tc.want)
			}
		})
	}
}

func TestGetURL(t *testing.T) {
	testCases := []struct {
		name       string
		httpStatus int
		response   string
		wantError  bool
		want       string
	}{
		{
			name:     "simple untar",
			response: `foo, bar`,
			want:     `foo, bar`,
		},
		{
			name:       "not found",
			httpStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(tc.response))

			var buf bytes.Buffer
			err := GetURL(server.URL, io.Writer(&buf))
			if tc.wantError == (err == nil) {
				t.Fatalf("GetURL(%q, buffer) got error: %v, want error? %v", server.URL, err, tc.wantError)
			}
			if tc.want != buf.String() {
				t.Errorf("GetURL(%q, buffer) = %v, want %v", server.URL, buf.String(), tc.want)
			}
		})
	}
}
