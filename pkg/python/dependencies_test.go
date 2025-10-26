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

package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestPackagePresent(t *testing.T) {
	testCases := []struct {
		name         string
		files        map[string]string
		releaseTrack string
		pkgName      string
		want         bool
	}{
		{
			name:    "requirements.txt_exists_and_contains_gunicorn",
			pkgName: "gunicorn",
			files: map[string]string{
				"requirements.txt": "gunicorn==19.9.0",
			},
			want: true,
		},
		{
			name:    "requirements.txt_exists_and_does_not_contain_gunicorn",
			pkgName: "gunicorn",
			files: map[string]string{
				"requirements.txt": "flask",
			},
			want: false,
		},
		{
			name:         "pyproject.toml_exists_and_contains_gunicorn_in_project.dependencies",
			pkgName:      "gunicorn",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["gunicorn>=20.1.0"]
				`,
			},
			want: true,
		},
		{
			name:         "pyproject.toml_exists_and_contains_gunicorn_in_project.dependencies_and_release_track_is_beta",
			pkgName:      "gunicorn",
			releaseTrack: "BETA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["gunicorn>=20.1.0"]
				`,
			},
			want: false,
		},
		{
			name:         "pyproject.toml_exists_and_contains_gunicorn_in_tool.poetry.dependencies",
			pkgName:      "gunicorn",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry]
					dependencies = { python = ">=3.9", gunicorn = ">=20.1.0" }
				`,
			},
			want: true,
		},
		{
			name:         "pyproject.toml_exists_and_does_not_contain_gunicorn",
			pkgName:      "gunicorn",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["requests>=2.0"]
				`,
			},
			want: false,
		},
		{
			name:         "Check_uvicorn_in_pyproject.toml",
			pkgName:      "uvicorn",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["fastapi>=0.111.0", "uvicorn>=0.30.1"]
				`,
			},
			want: true,
		},
		{
			name:         "Check_gradio_in_pyproject.toml",
			pkgName:      "gradio",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry.dependencies]
					python = ">=3.9"
					gradio = ">=4.0"
				`,
			},
			want: true,
		},
		{
			name:         "Check_gradio_not_present_in_pyproject.toml",
			pkgName:      "gradio",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry.dependencies]
					python = ">=3.9"
				`,
			},
			want: false,
		},
		{
			name:         "check_functions_framework_in_pyproject.toml",
			pkgName:      "functions-framework",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["functions-framework>=3.1.0"]
				`,
			},
			want: true,
		},
		{
			name:    "requirements.txt_exists_and_contains_google-adk",
			pkgName: "google-adk",
			files: map[string]string{
				"requirements.txt": "google-adk==0.1.0",
			},
			want: true,
		},
		{
			name:         "pyproject.toml_exists_and_contains_google-adk_in_project.dependencies",
			pkgName:      "google-adk",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[project]
					dependencies = ["google-adk>=0.1.0"]
				`,
			},
			want: true,
		},
		{
			name:         "pyproject.toml_exists_and_contains_google-adk_in_tool.poetry.dependencies",
			pkgName:      "google-adk",
			releaseTrack: "ALPHA",
			files: map[string]string{
				"pyproject.toml": `
					[tool.poetry.dependencies]
					google-adk = ">=0.1.0"
				`,
			},
			want: true,
		},
		{
			name:         "Neither_file_exists",
			pkgName:      "gunicorn",
			releaseTrack: "ALPHA",
			files:        map[string]string{},
			want:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current working directory: %v", err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("failed to change directory to %s: %v", dir, err)
			}

			defer os.Chdir(cwd)

			for path, content := range tc.files {
				fullPath := filepath.Join(dir, path)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("writing file %q: %v", fullPath, err)
				}
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))

			if tc.releaseTrack != "" {
				t.Setenv(env.ReleaseTrack, tc.releaseTrack)
			}

			got, err := PackagePresent(ctx, tc.pkgName)
			if err != nil {
				t.Fatalf("PackagePresent() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("PackagePresent() got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsFF(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "functions_framework_present",
			str:  "functions-framework==3.1.0\nflask\n",
			want: true,
		},
		{
			name: "functions_framework_present_with_comment",
			str:  "functions-framework #my-comment\nflask\n",
			want: true,
		},
		{
			name: "functions_framework_present_second_line",
			str:  "flask\nfunctions-framework==3.1.0",
			want: true,
		},
		{
			name: "no_functions_framework_present",
			str:  "functions-framework-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "functions_framework_egg_present",
			str:  "git+git://github.com/functions-framework@master#egg=functions-framework\nflask\n",
			want: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "functions-framework")
			if got != tc.want {
				t.Errorf("containsPackage(functions-framework) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsGunicorn(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gunicorn_present",
			str:  "gunicorn==19.9.0\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_with_comment",
			str:  "gunicorn #my-comment\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_second_line",
			str:  "flask\ngunicorn==19.9.0",
			want: true,
		},
		{
			name: "no_gunicorn_present",
			str:  "gunicorn-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_present",
			str:  "git+git://github.com/gunicorn@master#egg=gunicorn\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_egg_not_present",
			str:  "git+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nflask\n",
			want: false,
		},
		{
			name: "uvicorn_present",
			str:  "uvicorn==3.9.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_with_comment",
			str:  "uvicorn #my-comment\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_with_standard_version",
			str:  "uvicorn[standard] #my-comment\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0",
			want: false,
		},
		{
			name: "no_uvicorn_present",
			str:  "uvicorn-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present",
			str:  "uvicorn==3.9.0\ngunicorn==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\ngit+git://github.com/gunicorn@master#egg=gunicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\ngit+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0\ngunicorn==19.9.0",
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "gunicorn")
			if got != tc.want {
				t.Errorf("containsPackage(gunicorn) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsUvicorn(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gunicorn_present",
			str:  "gunicorn==19.9.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_present_with_comment",
			str:  "gunicorn #my-comment\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_present_second_line",
			str:  "flask\ngunicorn==19.9.0",
			want: false,
		},
		{
			name: "no_gunicorn_present",
			str:  "gunicorn-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_present",
			str:  "git+git://github.com/gunicorn@master#egg=gunicorn\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_not_present",
			str:  "git+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nflask\n",
			want: false,
		},
		{
			name: "uvicorn_present",
			str:  "uvicorn==3.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_with_comment",
			str:  "uvicorn #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_with_standard_version",
			str:  "uvicorn[standard] #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0",
			want: true,
		},
		{
			name: "no_uvicorn_present",
			str:  "uvicorn-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present",
			str:  "uvicorn==3.9.0\ngunicorn==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\ngit+git://github.com/gunicorn@master#egg=gunicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\ngit+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0\ngunicorn==19.9.0",
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "uvicorn")
			if got != tc.want {
				t.Errorf("containsPackage(uvicorn) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsGradio(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gradio_present",
			str:  "gradio==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_present_with_comment",
			str:  "gradio #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_present_second_line",
			str:  "fastapi\ngradio==19.9.0",
			want: true,
		},
		{
			name: "no_gradio_present",
			str:  "gradio-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "gradio_egg_present",
			str:  "git+git://github.com/gradio@master#egg=gradio\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_egg_not_present",
			str:  "git+git://github.com/gradio-logging@master#egg=gradio-logging\nfastapi\n",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "gradio")
			if got != tc.want {
				t.Errorf("containsPackage(gradio) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsStreamlit(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "streamlit_present",
			str:  "streamlit==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_present_with_comment",
			str:  "streamlit #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_present_second_line",
			str:  "fastapi\nstreamlit==19.9.0",
			want: true,
		},
		{
			name: "no_streamlit_present",
			str:  "streamlit-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "streamlit_egg_present",
			str:  "git+git://github.com/streamlit@master#egg=streamlit\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_egg_not_present",
			str:  "git+git://github.com/streamlit-logging@master#egg=streamlit-logging\nfastapi\n",
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "streamlit")
			if got != tc.want {
				t.Errorf("containsPackage(streamlit) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsFastAPIStandard(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "fastapi[standard]_present",
			str:  "fastapi[standard]",
			want: true,
		},
		{
			name: "fastapi[standard]_present_with_version",
			str:  "fastapi[standard]==0.1.0",
			want: true,
		},
		{
			name: "fastapi[standard]_present_with_comment",
			str:  "fastapi[standard] # a comment",
			want: true,
		},
		{
			name: "fastapi[standard]_present_second_line",
			str:  "gradio\nfastapi[standard]",
			want: true,
		},
		{
			name: "no_fastapi[standard]_present",
			str:  "fastapi[standard]-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "fastapi_only_present",
			str:  "fastapi",
			want: false,
		},
		{
			name: "fastapi_only_present_with_version",
			str:  "fastapi==0.1.0",
			want: false,
		},
		{
			name: "fastapi[standard]_egg_present",
			str:  "git+https://github.com/tiangolo/fastapi.git@master#egg=fastapi[standard]",
			want: true,
		},
		{
			name: "fastapi_egg_present",
			str:  "git+https://github.com/tiangolo/fastapi.git@master#egg=fastapi",
			want: false,
		},
		{
			name: "another_package_with_fastapi[standard]_as_substring",
			str:  "not-fastapi[standard]",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsPackage(tc.str, "fastapi[standard]"); got != tc.want {
				t.Errorf("containsPackage(%q, %q) = %v, want %v", tc.str, "fastapi[standard]", got, tc.want)
			}
		})
	}
}

func TestContainsGoogleAdk(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "google-adk_present",
			str:  "google-adk==0.1.0\nflask\n",
			want: true,
		},
		{
			name: "google-adk_present_with_comment",
			str:  "google-adk #my-comment\nflask\n",
			want: true,
		},
		{
			name: "google-adk_present_second_line",
			str:  "flask\ngoogle-adk==0.1.0",
			want: true,
		},
		{
			name: "no_google-adk_present",
			str:  "google-adk-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "google-adk_egg_present",
			str:  "git+git://github.com/google/adk@master#egg=google-adk\nflask\n",
			want: true,
		},
		{
			name: "google-adk_egg_not_present",
			str:  "git+git://github.com/google/adk-logging@master#egg=google-adk-logging\nflask\n",
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "google-adk")
			if got != tc.want {
				t.Errorf("containsPackage(google-adk) got %t, want %t", got, tc.want)
			}
		})
	}
}
