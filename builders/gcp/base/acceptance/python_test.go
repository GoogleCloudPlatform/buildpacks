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
package acceptance

import (
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptancePython(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:            "entrypoint from procfile web",
			App:             "simple",
			MustUse:         []string{pythonRuntime, pythonPIP, entrypoint},
			EnableCacheTest: true,
		},
		{
			Name:       "entrypoint from procfile custom",
			App:        "simple",
			Path:       "/custom",
			Entrypoint: "custom", // Must match the non-web process in Procfile.
			MustUse:    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "entrypoint from env",
			App:     "simple",
			Path:    "/custom",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "entrypoint with env var",
			App:     "simple",
			Path:    "/env?want=bar",
			Env:     []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "missing_entrypoint_main_py",
			App:     "missing_entrypoint_main_py",
			MustUse: []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
		},
		{
			Name:                       "missing_entrypoint_app_py",
			App:                        "missing_entrypoint_app_py",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.8.20"},
			VersionInclusionConstraint: "<3.9.0",
		},
		{
			Name:                       "missing_entrypoint_upgraded_app_py",
			App:                        "missing_entrypoint_upgraded_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.9.0",
		},
		{
			Name:                       "no_fastapi_smart_default_entrypoint_for_3.13_and_below",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			App:                        "fastapi_uvicorn",
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "fastapi_smart_default_entrypoint_for_3.13_and_above",
			App:                        "fastapi_uvicorn",
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_app_py_smart_default_entrypoint_for_3.13_and_above",
			App:                        "fastapi_uvicorn_app_py",
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_standard_smart_default_entrypoint_for_3.13_and_above",
			App:                        "fastapi_standard",
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_standard_app_py_smart_default_entrypoint_for_3.13_and_above",
			App:                        "fastapi_standard_app_py",
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_standard_smart_default_entrypoint_for_3.13_and_below",
			App:                        "fastapi_standard",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "fastapi_standard_app_py_smart_default_entrypoint_for_3.13_and_below",
			App:                        "fastapi_standard_app_py",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			Env:                        []string{"X_GOOGLE_FASTAPI_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "gradio_with_python_smart_defaults_for_3.13_and_above",
			App:                        "gradio",
			Env:                        []string{"X_GOOGLE_PYTHON_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "gradio_app_py_with_python_smart_defaults_for_3.13_and_above",
			App:                        "gradio_app_py",
			Env:                        []string{"X_GOOGLE_PYTHON_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "streamlit_with_python_smart_defaults_for_3.13_and_above",
			App:                        "streamlit",
			Env:                        []string{"X_GOOGLE_PYTHON_SMART_DEFAULTS=true", "GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
			MustMatch:                  "Streamlit",
		},
		{
			Name:       "runtime version 3.9",
			SkipStacks: []string{"google.24.full", "google.24"},
			App:        "version",
			Path:       "/version?want=3.9.16",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=3.9.16"},
			MustUse:    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:       "runtime version 3.8",
			SkipStacks: []string{"google.24.full", "google.24"},
			App:        "version",
			Path:       "/version?want=3.8.16",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=3.8.16"},
			MustUse:    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "runtime version from env",
			App:     "version",
			Path:    "/version?want=3.13.2",
			Env:     []string{"GOOGLE_RUNTIME_VERSION=3.13.2"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:       "runtime version from GOOGLE_PYTHON_VERSION env",
			SkipStacks: []string{"google.24.full", "google.24"},
			App:        "version",
			Path:       "/version?want=3.10.2",
			Env:        []string{"GOOGLE_RUNTIME_VERSION=3.10.2"},
			MustUse:    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:                       "runtime version from .python-version",
			VersionInclusionConstraint: "3.10.7", // version is set in the .python-version file
			SkipStacks:                 []string{"google.24.full", "google.24"},
			App:                        "python_version_file",
			Path:                       "/version?want=3.10.7",
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:    "python with client-side scripts correctly builds as a python app",
			App:     "scripts",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
		},
		{
			Name:                       "poetry_app",
			App:                        "poetry_app",
			MustUse:                    []string{pythonRuntime, pythonPoetry},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.9.0",
		},
		{
			Name:                       "poetry_main",
			App:                        "poetry_main",
			MustUse:                    []string{pythonRuntime, pythonPoetry},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.9.0",
		},
		{
			Name:                       "poetry_lock",
			App:                        "poetry_lock",
			MustUse:                    []string{pythonRuntime, pythonPoetry},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.9.0",
		},
		{
			Name:                       "poetry_setuptools",
			App:                        "poetry_setuptools",
			MustUse:                    []string{pythonRuntime, pythonPoetry},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.9.0",
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresPython(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "bad_runtime_version",
			App:       "simple",
			Env:       []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustMatch: "invalid Python version specified",
		},
		{
			Name:      "missing_main_and_app",
			App:       "missing_main_and_app",
			MustMatch: `.*for Python, provide a main.py or app.py file or set an entrypoint with "GOOGLE_ENTRYPOINT" env var or by creating a "Procfile" file`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
