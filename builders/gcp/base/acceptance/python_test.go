// Copyright 2020 Google LLC
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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptancePython(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "entrypoint from procfile web",
			App:             "python/simple",
			MustUse:         []string{pythonRuntime, pythonPIP, entrypoint},
			EnableCacheTest: true,
		},
		{
			Name:       "entrypoint from procfile custom",
			App:        "python/simple",
			Path:       "/custom",
			Entrypoint: "custom", // Must match the non-web process in Procfile.
			MustUse:    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "entrypoint from env",
			App:     "python/simple",
			Path:    "/custom",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "entrypoint with env var",
			App:     "python/simple",
			Path:    "/env?want=bar",
			Env:     []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "runtime version from env",
			App:     "python/version",
			Path:    "/version?want=3.8.0",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.8.0"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "runtime version from .python-version",
			App:     "python/version",
			Path:    "/version?want=3.8.1",
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:       "selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=python", "GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse:    []string{pythonRuntime},
			MustNotUse: []string{goRuntime, javaRuntime, nodeRuntime},
		},
		{
			Name:    "python with client-side scripts correctly builds as a python app",
			App:     "python/scripts",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
	}

	// Tests for specific versions of Python available on dl.google.com.
	for _, v := range []string{"3.7.12", "3.8.12", "3.9.10", "3.10.2"} {
		testCases = append(testCases, acceptance.Test{
			Name:    "dl.google.com runtime version " + v,
			App:     "python/version",
			Path:    "/version?want=" + v,
			Env:     []string{"GOOGLE_PYTHON_VERSION=" + v},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func TestFailuresPython(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad runtime version",
			App:       "python/simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustMatch: "invalid Python version specified",
		},
		{
			Name:      "python-version empty",
			App:       "python/empty_version",
			MustMatch: ".python-version exists but does not specify a version",
		},
		{
			Name:      "missing entrypoint",
			App:       "python/missing_entrypoint",
			MustMatch: `for Python, an entrypoint must be manually set, either with "GOOGLE_ENTRYPOINT" env var or by creating a "Procfile" file`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
