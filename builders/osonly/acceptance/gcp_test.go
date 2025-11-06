// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceOSOnly(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "simple_dart_app",
			App:     "simple_dart",
			MustUse: []string{dartCompile, dartSDK, dartPub},
		},
		{
			Name:    "simple_flutter_app",
			App:     "simple_flutter",
			MustUse: []string{dartCompile, dartSDK, dartPub},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresOSOnly(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad_runtime_version_dart",
			App:       "simple_dart",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "HTTP status: 404",
		},
		{
			Name:      "bad_runtime_version_flutter",
			App:       "simple_flutter",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS"},
			MustMatch: "version not found",
		},
	}

	for _, tc := range acceptance.FilterFailureTests(t, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
