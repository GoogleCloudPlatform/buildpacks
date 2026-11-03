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
	"os"
	"path/filepath"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "no env vars",
			want: 0,
		},
		{
			name: "GOOGLE_RUNTIME only",
			env:  []string{"GOOGLE_RUNTIME=python"},
			want: 0,
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH only",
			env:  []string{"X_GOOGLE_SKIP_RUNTIME_LAUNCH=true"},
			want: 0,
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH and GOOGLE_RUNTIME",
			env: []string{
				"X_GOOGLE_SKIP_RUNTIME_LAUNCH=true",
				"GOOGLE_RUNTIME=python39",
			},
			want: 0,
		},
		{
			name: "non-python GOOGLE_RUNTIME",
			env: []string{
				"X_GOOGLE_SKIP_RUNTIME_LAUNCH=true",
				"GOOGLE_RUNTIME=nodejs22",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestShouldLinkRuntime(t *testing.T) {
	testCases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "no env vars",
			want: false,
		},
		{
			name: "GOOGLE_RUNTIME only",
			env:  map[string]string{"GOOGLE_RUNTIME": "python313"},
			want: false,
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH only",
			env:  map[string]string{"X_GOOGLE_SKIP_RUNTIME_LAUNCH": "true"},
			want: false,
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH false",
			env: map[string]string{
				"X_GOOGLE_SKIP_RUNTIME_LAUNCH": "false",
				"GOOGLE_RUNTIME":               "python313",
			},
			want: false,
		},
		{
			name: "non-python GOOGLE_RUNTIME",
			env: map[string]string{
				"X_GOOGLE_SKIP_RUNTIME_LAUNCH": "true",
				"GOOGLE_RUNTIME":               "nodejs22",
			},
			want: false,
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH and python GOOGLE_RUNTIME",
			env: map[string]string{
				"X_GOOGLE_SKIP_RUNTIME_LAUNCH": "true",
				"GOOGLE_RUNTIME":               "python313",
			},
			want: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unsetEnv(t, "X_GOOGLE_SKIP_RUNTIME_LAUNCH")
			unsetEnv(t, "GOOGLE_RUNTIME")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			got, reason, err := shouldLinkRuntime()
			if err != nil {
				t.Fatalf("shouldLinkRuntime() failed: %v", err)
			}
			if got != tc.want {
				t.Errorf("shouldLinkRuntime() = %v (reason %q), want %v", got, reason, tc.want)
			}
		})
	}
}

func TestShouldLinkRuntimeInvalidEnv(t *testing.T) {
	t.Setenv("X_GOOGLE_SKIP_RUNTIME_LAUNCH", "notabool")
	t.Setenv("GOOGLE_RUNTIME", "python313")
	if _, _, err := shouldLinkRuntime(); err == nil {
		t.Error("shouldLinkRuntime() succeeded with invalid X_GOOGLE_SKIP_RUNTIME_LAUNCH, want error")
	}
}

func TestBuildNoOpWithoutEnv(t *testing.T) {
	testCases := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "no env vars",
		},
		{
			name: "GOOGLE_RUNTIME only",
			env:  map[string]string{"GOOGLE_RUNTIME": "python313"},
		},
		{
			name: "X_GOOGLE_SKIP_RUNTIME_LAUNCH only",
			env:  map[string]string{"X_GOOGLE_SKIP_RUNTIME_LAUNCH": "true"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unsetEnv(t, "X_GOOGLE_SKIP_RUNTIME_LAUNCH")
			unsetEnv(t, "GOOGLE_RUNTIME")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			// BuildFn must return before running python or touching the layer directory.
			if err := BuildFn(gcp.NewContext()); err != nil {
				t.Errorf("BuildFn() failed: %v", err)
			}
		})
	}
}

func TestLinkDirsTo(t *testing.T) {
	layerPath := t.TempDir()
	pythonPath := t.TempDir()
	for _, d := range linkDirs {
		if err := os.MkdirAll(filepath.Join(layerPath, d, "subdir"), 0755); err != nil {
			t.Fatalf("creating layer dir %q: %v", d, err)
		}
	}

	if err := linkDirsTo(layerPath, pythonPath); err != nil {
		t.Fatalf("linkDirsTo(%q, %q) failed: %v", layerPath, pythonPath, err)
	}

	for _, d := range linkDirs {
		p := filepath.Join(layerPath, d)
		got, err := os.Readlink(p)
		if err != nil {
			t.Fatalf("reading link %q: %v", p, err)
		}
		if want := filepath.Join(pythonPath, d); got != want {
			t.Errorf("link %q points to %q, want %q", p, got, want)
		}
	}
}

// unsetEnv unsets the env var key for the duration of the test.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "") // Restores the original value on cleanup.
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetting %s: %v", key, err)
	}
}
