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

const (
	entrypoint              = "google.config.entrypoint"
	pythonFF                = "google.python.functions-framework"
	pythonPIP               = "google.python.pip"
	pythonRuntime           = "google.python.runtime"
	pythonMissingEntrypoint = "google.python.missing-entrypoint"
	pythonWebserver         = "google.python.webserver"
	pythonPoetry            = "google.python.poetry"
	pythonUV                = "google.python.uv"
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
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_web_upgraded",
			App:                        "simple_new",
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			EnableCacheTest:            true,
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "entrypoint_from_procfile_web_upgraded_default_uv",
			App:                        "simple_new",
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			EnableCacheTest:            true,
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "entrypoint_from_procfile_web_upgraded_uv",
			App:                        "simple_new",
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			EnableCacheTest:            true,
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom",
			App:                        "simple",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom_upgraded",
			App:                        "simple_new",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom_upgraded_default_uv",
			App:                        "simple_new",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "entrypoint_from_procfile_custom_upgraded_uv",
			App:                        "simple_new",
			Path:                       "/custom",
			Entrypoint:                 "custom", // Must match the non-web process in Procfile.
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "entrypoint_from_env",
			App:                        "simple",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "entrypoint_from_env_upgraded",
			App:                        "simple_new",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "entrypoint_from_env_upgraded_default_uv",
			App:                        "simple_new",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "entrypoint_from_env_upgraded_uv",
			App:                        "simple_new",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "entrypoint_from_env_upgraded_pip",
			App:                        "simple_new",
			Path:                       "/custom",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 custom:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "entrypoint_with_env_var",
			App:                        "simple",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded_default_uv",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded_uv",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "entrypoint_with_env_var_upgraded_pip",
			App:                        "simple_new",
			Path:                       "/env?want=bar",
			Env:                        []string{"GOOGLE_ENTRYPOINT=FOO=bar gunicorn -b :8080 main:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app",
			App:                        "scripts",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app_upgraded",
			App:                        "scripts_new",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app_upgraded_default_uv",
			App:                        "scripts_new",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "python_with_client-side_scripts_correctly_builds_as_a_python_app_upgraded_uv",
			App:                        "scripts_new",
			Env:                        []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.8.0",
		},
		{
			Name:    "python_module_dependency_using_a_native_extension",
			App:     "native_extensions",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
			// numpy 1.23.1 requires Python 3.8 and <3.12.0.
			VersionInclusionConstraint: ">=3.8.0 <3.12.0",
		},
		{
			Name:    "python_module_dependency_using_a_native_extension_for_3.10_and_above",
			App:     "native_extensions_above_python310",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app"},
			MustUse: []string{pythonRuntime, pythonPIP, entrypoint},
			// numpy 2.1 needed to support 3.13 only works on python 3.10 and above.
			VersionInclusionConstraint: ">= 3.10.0 < 3.14.0",
		},
		{
			Name:    "python_module_dependency_using_a_native_extension_for_3.10_and_above_default_uv",
			App:     "native_extensions_above_python310",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse: []string{pythonRuntime, pythonUV, entrypoint},
			// numpy 2.1 needed to support 3.13 only works on python 3.10 and above.
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:    "python_module_dependency_using_a_native_extension_for_3.10_and_above_uv",
			App:     "native_extensions_above_python310",
			Env:     []string{"GOOGLE_ENTRYPOINT=gunicorn -b :8080 main:app", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse: []string{pythonRuntime, pythonUV, entrypoint},
			// numpy 2.1 needed to support 3.13 only works on python 3.10 and above.
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "requirements_vendored_dependencies",
			App:                        "requirements_vendored_dependencies",
			Env:                        []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=package"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.10.0 < 3.14.0",
		},
		{
			Name:                       "requirements_vendored_dependencies_default_uv",
			App:                        "requirements_vendored_dependencies",
			Env:                        []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=package"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "requirements_vendored_dependencies_uv",
			App:                        "requirements_vendored_dependencies",
			Env:                        []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=package", "X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_vendored_dependencies_pip",
			App:                        "pyproject_vendored_dependencies",
			Env:                        []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=package", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPIP, entrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_vendored_dependencies_uv",
			App:                        "pyproject_vendored_dependencies",
			Env:                        []string{"GOOGLE_VENDOR_PIP_DEPENDENCIES=package", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, entrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "uvicorn_3.13_and_above",
			App:                        "fastapi_uvicorn",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "uvicorn_3.13_and_above_uv",
			App:                        "fastapi_uvicorn",
			Env:                        []string{"GOOGLE_PYTHON_PACKAGE_MANAGER=uv", "X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "uvicorn_app_py_3.13_and_above",
			App:                        "fastapi_uvicorn_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "uvicorn_3.13_and_below",
			App:                        "fastapi_uvicorn",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "uvicorn_3.13_and_below_app_py",
			App:                        "fastapi_uvicorn_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "fastapi_standard_3.13_and_above",
			App:                        "fastapi_standard",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "fastapi_standard_app_py_3.13_and_above",
			App:                        "fastapi_standard_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "fastapi_standard_3.13_and_below",
			App:                        "fastapi_standard",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "fastapi_standard_3.13_and_below_app_py",
			App:                        "fastapi_standard_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: "<3.13.0",
			MustMatchStatusCode:        http.StatusInternalServerError,
			MustMatch:                  "Internal Server Error",
		},
		{
			Name:                       "gradio_3.13_and_above",
			App:                        "gradio",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "gradio_app_py_3.13_and_above",
			App:                        "gradio_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "streamlit_3.13_and_above",
			App:                        "streamlit",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
			MustMatch:                  "Streamlit",
		},
		{
			Name:                       "missing_entrypoint_main_py",
			App:                        "missing_entrypoint_main_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "< 3.9.0",
		},
		{
			Name:                       "missing_entrypoint_main_py_upgraded",
			App:                        "missing_entrypoint_main_py_new",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "missing_entrypoint_main_py_upgraded_default_uv",
			App:                        "missing_entrypoint_main_py_new",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "missing_entrypoint_app_py",
			App:                        "missing_entrypoint_app_py",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: "< 3.8.0",
		},
		{
			Name:                       "missing_entrypoint_app_py_upgraded",
			App:                        "missing_entrypoint_app_py_new",
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.8.0 < 3.14.0",
		},
		{
			Name:                       "missing_entrypoint_app_py_upgraded_default_uv",
			App:                        "missing_entrypoint_app_py_new",
			MustUse:                    []string{pythonRuntime, pythonUV, pythonWebserver, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "poetry_app",
			App:                        "poetry_app",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_main",
			App:                        "poetry_main",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_lock",
			App:                        "poetry_lock",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.10.0",
		},
		{
			Name:                       "poetry_setuptools",
			App:                        "poetry_setuptools",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.10.0 <3.14.0",
		},
		{
			Name:                       "uv_main",
			App:                        "uv_main",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "uv_app",
			App:                        "uv_app",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_without_lock",
			App:                        "pyproject_without_lock",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_without_lock_pip",
			App:                        "pyproject_without_lock",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_without_lock_uv",
			App:                        "pyproject_without_lock",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=uv"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "poetry_uvicorn",
			App:                        "poetry_uvicorn",
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			VersionInclusionConstraint: ">=3.13.0 <3.14.0",
		},
		{
			Name:                       "pyproject_uvicorn",
			App:                        "pyproject_uvicorn",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_uvicorn_pip",
			App:                        "pyproject_uvicorn",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_fastapi_standard",
			App:                        "pyproject_fastapi_standard",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_fastapi_standard_pip",
			App:                        "pyproject_fastapi_standard",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "poetry_gradio",
			App:                        "poetry_gradio",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_gradio",
			App:                        "pyproject_gradio",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_gradio_pip",
			App:                        "pyproject_gradio",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_streamlit",
			App:                        "pyproject_streamlit",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
			MustMatch:                  "Streamlit",
		},
		{
			Name:                       "pyproject_streamlit_pip",
			App:                        "pyproject_streamlit",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
			MustMatch:                  "Streamlit",
		},
		{
			Name:                       "pyproject_script",
			App:                        "pyproject_script",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_script_pip",
			App:                        "pyproject_script",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA", "GOOGLE_PYTHON_PACKAGE_MANAGER=pip"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0 < 3.14.0",
		},
		{
			Name:                       "poetry_script",
			App:                        "poetry_script",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPoetry, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0",
		},
		{
			Name:                       "pyproject_and_requirements_default_pip",
			App:                        "pyproject_and_requirements",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.10.0 < 3.14.0",
		},
		{
			Name:                       "pyproject_and_requirements_default_uv",
			App:                        "pyproject_and_requirements",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.14.0",
		},
		{
			Name:                       "google_adk_in_requirements_txt",
			App:                        "google_adk",
			Path:                       "/list-apps",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonPIP, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
			MustMatch:                  "multi_tool_agent",
		},
		{
			Name:                       "google_adk_in_pyproject_toml",
			App:                        "pyproject_google_adk",
			Path:                       "/list-apps",
			Env:                        []string{"X_GOOGLE_RELEASE_TRACK=ALPHA"},
			MustUse:                    []string{pythonRuntime, pythonUV, pythonMissingEntrypoint},
			VersionInclusionConstraint: ">= 3.13.0 < 3.14.0",
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
			Name:      "bad runtime version",
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
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}
