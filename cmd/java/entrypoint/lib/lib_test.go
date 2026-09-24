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
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
	"github.com/buildpacks/libcnb/v2"
)

func TestDetect(t *testing.T) {
	// The buildpack always opts in.
	buildpacktest.TestDetect(t, DetectFn, "no files", map[string]string{}, []string{}, 0)
}

func TestBuildFn(t *testing.T) {
	testCases := []struct {
		name    string
		jars    map[string]string // rel_path -> manifest content
		devmode bool
		wantCmd []string
	}{
		{
			name: "relative_jar_in_target",
			jars: map[string]string{
				"target/app.jar": "Main-Class: com.example.Main\n",
			},
			wantCmd: []string{"java", "-jar", "target/app.jar"},
		},
		{
			name: "relative_jar_in_root",
			jars: map[string]string{
				"app.jar": "Main-Class: com.example.Main\n",
			},
			wantCmd: []string{"java", "-jar", "app.jar"},
		},
		{
			name: "devmode_relative_jar",
			jars: map[string]string{
				"target/app.jar": "Main-Class: com.example.Main\n",
			},
			devmode: true,
			wantCmd: []string{"java", "-jar", "target/app.jar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			for relPath, mf := range tc.jars {
				jarPath := filepath.Join(appDir, relPath)
				if err := os.MkdirAll(filepath.Dir(jarPath), 0755); err != nil {
					t.Fatalf("Failed to create dir: %v", err)
				}
				createJar(t, jarPath, mf)
			}

			if tc.devmode {
				t.Setenv("GOOGLE_DEVMODE", "true")
			}

			layersDir := t.TempDir()
			ctx := gcp.NewContext(
				gcp.WithApplicationRoot(appDir),
				gcp.WithBuildContext(libcnb.BuildContext{Layers: libcnb.Layers{Path: layersDir}}),
			)
			if err := BuildFn(ctx); err != nil {
				t.Fatalf("BuildFn() failed: %v", err)
			}

			if !tc.devmode {
				processes := ctx.Processes()
				if len(processes) != 1 {
					t.Fatalf("expected 1 process, got %d", len(processes))
				}
				if diff := cmp.Diff(tc.wantCmd, processes[0].Command); diff != "" {
					t.Errorf("process command mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func createJar(t *testing.T, path, mfContent string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create jar %s: %v", path, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	if mfContent != "" {
		mf, err := zw.Create("META-INF/MANIFEST.MF")
		if err != nil {
			t.Fatalf("Failed to create META-INF/MANIFEST.MF: %v", err)
		}
		if _, err := mf.Write([]byte(mfContent)); err != nil {
			t.Fatalf("Failed to write to META-INF/MANIFEST.MF: %v", err)
		}
	}
}
