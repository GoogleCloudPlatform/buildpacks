// Copyright 2025 Google LLC
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

package lib

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "no_py_files",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
		{
			name: "has_main_py_file",
			files: map[string]string{
				"main.py": "",
			},
			want: 0,
		},
		{
			name: "has_multiple_py_files",
			files: map[string]string{
				"main.py":  "",
				"app.py":   "",
				"utils.py": "",
			},
			want: 0,
		},
		{
			name: "has_multiple_py_files_with_nested_directory",
			files: map[string]string{
				"folder/main.py":     "",
				"folder/__init__.py": "",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name         string
		files        map[string]string
		env          []string
		wantCmd      []string
		runtime      string
		wantExitCode int
	}{
		{
			name: "default_gunicorn",
			files: map[string]string{
				"main.py": "",
			},
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_fastapi_standard_with_gunicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi[standard]\ngunicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_fastapi_standard_with_uvicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi[standard]\nuvicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_gunicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "gunicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_uvicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "uvicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_fastapi_standard",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi[standard]",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_fastapi_standard_below_3.13",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi[standard]",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.12.0",
			},
			runtime: "python3.12",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_fastapi_standard_no_version",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi[standard]",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
			},
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_fastapi",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_fastapi_with_gunicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi\ngunicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_fastapi_with_uvicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "fastapi\nuvicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_gradio",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "gradio",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"python", "main.py"},
		},
		{
			name: "python_smart_defaults_streamlit",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "streamlit",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"streamlit", "run", "main.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
		},
		{
			name: "python_smart_defaults_none",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_below_3.13_uvicorn",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.12.0",
			},
			runtime: "python3.12",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_below_3.13_gradio",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "gradio",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.12.0",
			},
			runtime: "python3.12",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_below_3.13_streamlit",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "streamlit",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.12.0",
			},
			runtime: "python3.12",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_uvicorn_with_no_version",
			files: map[string]string{
				"main.py":          "",
				"requirements.txt": "uvicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
			},
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "default_gunicorn_app_py",
			files: map[string]string{
				"app.py": "",
			},
			wantCmd: []string{"gunicorn", "-b", ":8080", "app:app"},
		},
		{
			name: "default_gunicorn_main_py_and_app_py",
			files: map[string]string{
				"main.py": "",
				"app.py":  "",
			},
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "python_smart_defaults_gunicorn_app_py",
			files: map[string]string{
				"app.py":           "",
				"requirements.txt": "gunicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "app:app"},
		},
		{
			name: "python_smart_defaults_uvicorn_app_py",
			files: map[string]string{
				"app.py":           "",
				"requirements.txt": "uvicorn",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "app:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "python_smart_defaults_gradio_app_py",
			files: map[string]string{
				"app.py":           "",
				"requirements.txt": "gradio",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"python", "app.py"},
		},
		{
			name: "python_smart_defaults_streamlit_app_py",
			files: map[string]string{
				"app.py":           "",
				"requirements.txt": "streamlit",
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"streamlit", "run", "app.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
		},
		{
			name: "pyproject_gunicorn",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["gunicorn"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "pyproject_uvicorn",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["fastapi", "uvicorn"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "pyproject_fastapi_standard",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["fastapi[standard]"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "pyproject_gradio",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["gradio"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"python", "main.py"},
		},
		{
			name: "pyproject_streamlit",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["streamlit"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"streamlit", "run", "main.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
		},
		{
			name: "pyproject_poetry_gunicorn",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[tool.poetry.dependencies]
gunicorn = "*_*"`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "pyproject_poetry_uvicorn",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[tool.poetry.dependencies]
uvicorn = "*_*"`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "pyproject_poetry_fastapi_standard",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[tool.poetry.dependencies]
"fastapi[standard]" = "*"`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"uvicorn", "main:app", "--port", "8080", "--host", "0.0.0.0"},
		},
		{
			name: "pyproject_poetry_gradio",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[tool.poetry.dependencies]
gradio = "*_*"`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"python", "main.py"},
		},
		{
			name: "pyproject_poetry_streamlit",
			files: map[string]string{
				"app.py": "",
				"pyproject.toml": `[tool.poetry.dependencies]
streamlit = "*_*"`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"streamlit", "run", "app.py", "--server.address", "0.0.0.0", "--server.port", "8080"},
		},
		{
			name: "pyproject_no_webserver",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project]
dependencies = ["requests"]`,
			},
			env: []string{
				env.PythonSmartDefaults + "=true",
				env.ReleaseTrack + "=ALPHA",
				env.RuntimeVersion + "=3.13.0",
			},
			runtime: "python3.13",
			wantCmd: []string{"gunicorn", "-b", ":8080", "main:app"},
		},
		{
			name: "pyproject_script_uv",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[project.scripts]
any_start_cmd = "main:app"`,
			},
			env: []string{
				env.ReleaseTrack + "=ALPHA",
			},
			wantCmd: []string{"uv", "run", "any_start_cmd"},
		},
		{
			name: "pyproject_script_poetry",
			files: map[string]string{
				"main.py": "",
				"pyproject.toml": `[tool.poetry]
name = "test"
version = "1.2.3"
description = ""
authors = ["test"]
[tool.poetry.scripts]
any_start_cmd = "main:app"`,
			},
			env: []string{
				env.ReleaseTrack + "=ALPHA",
			},
			wantCmd: []string{"poetry", "run", "any_start_cmd"},
		},
		{
			name: "no_main_or_app",
			files: map[string]string{
				"other.py": "",
			},
			wantExitCode: 1,
		},
		{
			name: "main_in_nested_directory",
			files: map[string]string{
				"folder/main.py": "",
			},
			wantExitCode: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithEnvs(tc.env...),
			}
			result, err := bpt.RunBuild(t, BuildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}
			wantCommand := strings.Join(tc.wantCmd, " ")
			if result.ExitCode == 0 && !processAdded(result, wantCommand) {
				t.Errorf("expected command %q to be added as Process, but it was not, build output: %s", wantCommand, result.Output)
			}
		})
	}
}

// ProcessAdded returns the true if the process added to the context.
func processAdded(r *bpt.Result, command string) bool {
	re := regexp.MustCompile(fmt.Sprintf(`(?s)Setting default entrypoint: .*?%s`, command))
	return re.FindString(r.Output) != ""
}
