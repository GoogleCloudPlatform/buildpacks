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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

const (
	latestVersionPath = "latest.version"
	releaseIndexURL   = "releases-index.json"
)

func TestGetRuntimeVersionForSDKVersion(t *testing.T) {
	testCases := []struct {
		Name                         string
		SdkVersion                   string
		ExpectedRuntimeVersionResult string
		ExpectedErrMsg               string
	}{
		{
			Name:                         "Happy Path",
			SdkVersion:                   "3.1.302",
			ExpectedRuntimeVersionResult: "3.1.6",
		},
		{
			Name:           "unknown release channel",
			SdkVersion:     "3.45.0",
			ExpectedErrMsg: `finding release index match for "3.45.0": no match`,
		},
		{
			Name:           "unknown sdk version",
			SdkVersion:     "3.1.500",
			ExpectedErrMsg: `finding release for sdk "3.1.500" in channel "3.1": no match`,
		},
	}

	addr, cleanup := startMockReleaseService(t)
	defer cleanup()
	releaseClient := newMockReleaseClient(addr)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rtVersion, err := GetRuntimeVersionForSDKVersion(releaseClient, tc.SdkVersion)
			if err == nil && tc.ExpectedErrMsg != "" {
				t.Errorf("error from GetRuntimeVersionForSDKVersion(client, %v) = nil, want %v", tc.SdkVersion, tc.ExpectedErrMsg)
			}
			if err != nil && tc.ExpectedErrMsg != err.Error() {
				t.Errorf("error from GetRuntimeVersionForSDKVersion(client, %v) = %q, want %q", tc.SdkVersion, err.Error(), tc.ExpectedErrMsg)
			}
			if rtVersion != tc.ExpectedRuntimeVersionResult {
				t.Errorf("GetRuntimeVersionForSDKVersion(client, %v) = %q, want %q", tc.SdkVersion, rtVersion, tc.ExpectedRuntimeVersionResult)
			}
		})
	}
}

func startMockReleaseService(t *testing.T) (string, func()) {
	t.Helper()
	handler := customHandler{}
	server := httptest.NewServer(&handler)
	handler.baseURL = fmt.Sprintf("%s/testdata", server.URL)
	return handler.baseURL, server.Close
}

type customHandler struct {
	// baseURL replaces the "base URL" of URLs include in the response so that subsquent requests will be made
	// to the local test web service.
	baseURL string
}

func (h *customHandler) ServeHTTP(respWriter http.ResponseWriter, request *http.Request) {
	value, err := h.readTestDataAndSubstituteVariables(request)
	if err == nil {
		respWriter.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(respWriter, value); err != nil {
			http.Error(respWriter, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(respWriter, "404 page not found", http.StatusNotFound)
		} else {
			http.Error(respWriter, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *customHandler) readTestDataAndSubstituteVariables(request *http.Request) (string, error) {
	testdataPath := testdata.MustGetPath(request.URL.Path)
	bytes, err := os.ReadFile(testdataPath)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(bytes), "${baseurl}", h.baseURL), nil
}

func newMockReleaseClient(baseURL string) *client.Client {
	return client.NewWithEndpoints(fmt.Sprintf("%v/%v", baseURL, latestVersionPath), fmt.Sprintf("%v/%v", baseURL, releaseIndexURL))
}
