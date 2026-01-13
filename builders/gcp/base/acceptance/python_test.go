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
			Name:                       "entrypoint_from_procfile_web",
			App:                        "simple",
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			EnableCacheTest:            true,
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "entrypoint_from_procfile_web_upgraded_app",
			App:                        "simple_new",
			MustUse:                    []string{pythonRuntime, entrypoint},
			EnableCacheTest:            true,
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_web_upgraded_app_uv",
			App:                        "simple_new",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			EnableCacheTest:            true,
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom",
			App:                        "simple",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "entrypoint_from_procfile_custom_upgraded_app",
			App:                        "simple_new",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, entrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom_upgraded_app_uv",
			App:                        "simple_new",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "entrypoint_from_env",
			App:                        "simple",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "entrypoint_from_env_upgraded_app",
			App:                        "simple_new",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse:                    []string{pythonRuntime, entrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "entrypoint_with_env_var",
			App:                        "simple",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded_app",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">=3.8.0 < 3.14.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded_app_uv",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "missing_entrypoint_main_py",
			App:                        "missing_entrypoint_main_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "missing_entrypoint_main_py_new",
			App:                        "missing_entrypoint_main_py_new",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.8.0 < 3.14.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "missing_entrypoint_app_py",
			App:                        "missing_entrypoint_app_py",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.8.20"},
			VersionInclusionConstraint: "<3.8.0",
		},
		{
			Name:                       "missing_entrypoint_app_py_new",
			App:                        "missing_entrypoint_app_py_new",
			MustUse:                    []string{pythonRuntime, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "uvicorn_3.13_and_below",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			App:                        "fastapi_uvicorn",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "uvicorn_3.13_and_above",
			App:                        "fastapi_uvicorn",
			MustUse:                    []string{pythonRuntime, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "uvicorn_app_py_3.13",
			App:                        "fastapi_uvicorn_app_py",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "uvicorn_app_py_3.14_and_above",
			App:                        "fastapi_uvicorn_app_py",
			SkipStacks:                 []string{"google.gae.22", "google.22"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.14.0",
		},
		{
			Name:                       "fastapi_standard_3.13_and_above",
			App:                        "fastapi_standard",
			MustUse:                    []string{pythonRuntime, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_standard_app_py_3.13_and_above",
			App:                        "fastapi_standard_app_py",
			MustUse:                    []string{pythonRuntime, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "fastapi_standard_3.13_and_below",
			App:                        "fastapi_standard",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "fastapi_standard_app_py_3.13_and_below",
			App:                        "fastapi_standard_app_py",
			SkipStacks:                 []string{"google.24.full", "google.24"},
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.12.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "gradio_3.13_and_above",
			App:                        "gradio",
			MustUse:                    []string{pythonRuntime, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "gradio_app_py_3.13_and_above",
			App:                        "gradio_app_py",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=3.13.0"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "gradio_app_py_3.13_and_above_default_uv",
			App:                        "gradio_app_py",
			SkipStacks:                 []string{"google.gae.22", "google.22"},
			MustUse:                    []string{pythonRuntime, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.14.0",
		},
		{
			Name:                       "streamlit_3.13_and_above",
			App:                        "streamlit",
			MustUse:                    []string{pythonRuntime, pythonMissingEntrypoint},
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
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app",
			App:                        "scripts",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "<3.8.0",
			SkipStacks:                 []string{"google.24.full", "google.24"},
		},
		{
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app_(upgraded)",
			App:                        "scripts_new",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, entrypoint},
			VersionInclusionConstraint: ">=3.8.0",
		},
		{
			Name:                       "poetry_app",
			App:                        "poetry_app",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_main",
			App:                        "poetry_main",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_lock",
			App:                        "poetry_lock",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_setuptools",
			App:                        "poetry_setuptools",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "uv_app",
			App:                        "uv_app",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "uv_main",
			App:                        "uv_main",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "pyproject_without_lock",
			App:                        "pyproject_without_lock",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "pyproject_without_lock_pip",
			App:                        "pyproject_without_lock",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "pyproject_without_lock_uv",
			App:                        "pyproject_without_lock",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_uvicorn",
			App:                        "poetry_uvicorn",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "pyproject_uvicorn",
			App:                        "pyproject_uvicorn",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "pyproject_uvicorn_pip",
			App:                        "pyproject_uvicorn",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "pyproject_fastapi_standard",
			App:                        "pyproject_fastapi_standard",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "pyproject_fastapi_standard_pip",
			App:                        "pyproject_fastapi_standard",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "poetry_gradio",
			App:                        "poetry_gradio",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0",
		},
		{
			Name:                       "pyproject_gradio",
			App:                        "pyproject_gradio",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "pyproject_gradio_pip",
			App:                        "pyproject_gradio",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
		},
		{
			Name:                       "pyproject_streamlit",
			App:                        "pyproject_streamlit",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
			MustMatch:                  "Streamlit",
		},
		{
			Name:                       "pyproject_script",
			App:                        "pyproject_script",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_script_pip",
			App:                        "pyproject_script",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "poetry_script",
			App:                        "poetry_script",
			Env:                        []string{},
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "google_adk_in_requirements_txt",
			App:                        "google_adk",
			Path:                       "/list-apps",
			MustUse:                    []string{pythonRuntime, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
			MustMatch:                  "multi_tool_agent",
		},
		{
			Name:                       "google_adk_in_pyproject_toml",
			App:                        "pyproject_google_adk",
			Path:                       "/list-apps",
			MustUse:                    []string{pythonRuntime, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0",
			MustMatch:                  "multi_tool_agent",
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func TestFailuresPython(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:                       "bad_runtime_version",
			App:                        "simple",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustMatch:                  "invalid Python version specified",
			VersionInclusionConstraint: "<3.8.0",
		},
		{
			Name:                       "bad_runtime_version_upgraded_app",
			App:                        "simple_new",
			Env:                        []string{"GOOGLE_RUNTIME_VERSION=BAD_NEWS_BEARS", "GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustMatch:                  "invalid Python version specified",
			VersionInclusionConstraint: ">=3.8.0",
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
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
