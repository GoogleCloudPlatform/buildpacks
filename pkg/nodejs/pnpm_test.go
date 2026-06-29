// Copyright 2023 Google LLC
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
package nodejs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/tooling"
	"github.com/buildpacks/libcnb/v2"
)

func TestInstallPNPM(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		wantFile    string
		wantError   bool
	}{
		{
			name:     "no_version_constraint",
			wantFile: "bin/pnpm",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "11.0.0"
				},
				"versions": {
					"11.0.0": {
						"name": "npm",
						"version": "11.0.0"
					}
				},
				"modified": "2026-05-21T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{},
		},
		{
			name:     "valid_version_constraint",
			wantFile: "bin/pnpm",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.x.x",
				},
			},
		},
		{
			name: "invalid_version",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "8.4.0"
				},
				"versions": {
					"8.4.0": {
						"name": "npm",
						"version": "8.4.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: ">9.0.0",
				},
			},
			wantError: true,
		},
		{
			// Regression test for https://github.com/GoogleCloudPlatform/buildpacks/issues/627:
			// a range constraint like ">=9.0.0" resolves to the highest matching version (v11),
			// which must install via the tarball path rather than 404 on a missing naked binary.
			name:     "version_range_resolves_to_v11",
			wantFile: "bin/pnpm",
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "11.0.0"
				},
				"versions": {
					"9.15.0": {
						"name": "pnpm",
						"version": "9.15.0"
					},
					"11.0.0": {
						"name": "pnpm",
						"version": "11.0.0"
					}
				}
			}`,
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: ">=9.0.0",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.String(), ".tar.gz") {
						w.WriteHeader(http.StatusOK)
						w.Write(mockTarballBytes(t))
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("pnpm!"))
					}
				})),
				testserver.WithMockURL(&pnpmDownloadURL),
			)
			testserver.New(
				t,
				testserver.WithJSON(tc.npmResponse),
				testserver.WithMockURL(&npmRegistryURL),
			)

			layer := &libcnb.Layer{
				Name:     "pnpm_test",
				Path:     t.TempDir(),
				Metadata: map[string]any{},
			}
			err := InstallPNPM(gcpbuildpack.NewContext(), layer, &tc.packageJSON)
			if tc.wantError == (err == nil) {
				t.Fatalf("InstallPNPM() got error: %v, want error? %v", err, tc.wantError)
			}

			if tc.wantFile != "" {
				fp := filepath.Join(layer.Path, tc.wantFile)
				if _, err := os.Stat(fp); err != nil {
					t.Errorf("Missing file: %s (%v)", fp, err)
				}
			}
		})
	}
}
func TestDetectPNPMVersion(t *testing.T) {
	testCases := []struct {
		name        string
		npmResponse string
		packageJSON PackageJSON
		stackID     string
		wantVersion string
		wantError   bool
	}{
		{
			name:        "no_package.json_returns_pinned_version_from_tooling_bzl",
			packageJSON: PackageJSON{},
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "9.2.0"
				},
				"versions": {
					"9.2.0": {
						"name": "npm",
						"version": "9.2.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			stackID:     "ubuntu1804",
			wantVersion: "10.12.4", // pinned version for ubuntu1804
		},
		{
			name:        "no_package.json_returns_latest_version_from_tooling_bzl",
			packageJSON: PackageJSON{},
			npmResponse: `{
				"name": "pnpm",
				"dist-tags": {
					"latest": "9.2.0"
				},
				"versions": {
					"9.2.0": {
						"name": "npm",
						"version": "9.2.0"
					}
				},
				"modified": "2022-01-27T21:10:55.626Z"
			}`,
			stackID:     "ubuntu2204",
			wantVersion: "10.32.1",
		},
		{
			name: "only_engines_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.2.0",
				},
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "only_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "pnpm@8.2.0",
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "both_engine_and_packageManager_version",
			packageJSON: PackageJSON{
				Engines: packageEnginesJSON{
					PNPM: "8.2.0",
				},
				PackageManager: "pnpm@8.1.0",
			},
			stackID:     "ubuntu2204",
			wantVersion: "8.2.0",
		},
		{
			name: "invalid_packageManager_version",
			packageJSON: PackageJSON{
				PackageManager: "yarn@8.2.0",
			},
			stackID:   "ubuntu2204",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithJSON(tc.npmResponse),
				testserver.WithMockURL(&npmRegistryURL),
			)

			ctx := gcpbuildpack.NewContext()
			defer tooling.MockData()()

			version, err := detectPNPMVersion(ctx, &tc.packageJSON, tc.stackID)
			if version != tc.wantVersion {
				t.Errorf("detectPNPMVersion() got version: %v, want version: %v", version, tc.wantVersion)
			}
			if tc.wantError == (err == nil) {
				t.Fatalf("detectPNPMVersion() got error: %v, want error? %v", err, tc.wantError)
			}
		})
	}
}

func TestInstallPNPMV11(t *testing.T) {
	npmResponse := `{
		"name": "pnpm",
		"dist-tags": {
			"latest": "11.0.0"
		},
		"versions": {
			"11.0.0": {
				"name": "pnpm",
				"version": "11.0.0"
			}
		}
	}`

	testserver.New(
		t,
		testserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.String(), ".tar.gz") {
				w.WriteHeader(http.StatusOK)
				w.Write(mockTarballBytes(t))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		})),
		testserver.WithMockURL(&pnpmDownloadURL),
	)

	testserver.New(
		t,
		testserver.WithJSON(npmResponse),
		testserver.WithMockURL(&npmRegistryURL),
	)

	layer := &libcnb.Layer{
		Name:     "pnpm_test",
		Path:     t.TempDir(),
		Metadata: map[string]any{},
	}

	pkgJSON := &PackageJSON{
		Engines: packageEnginesJSON{
			PNPM: "11.0.0",
		},
	}

	err := InstallPNPM(gcpbuildpack.NewContext(), layer, pkgJSON)
	if err != nil {
		t.Fatalf("InstallPNPM(ctx, %v, %+v) got error: %v, want nil", layer, pkgJSON, err)
	}

	// Verify the native binary was replaced with the bash wrapper that uses node to execute
	// dist/pnpm.mjs, avoiding a libatomic dependency on minimal run images.
	fp := filepath.Join(layer.Path, "bin/pnpm")
	content, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) got error: %v, want nil", fp, err)
	}
	wantWrapper := "#!/usr/bin/env bash\nexec node \"$(dirname \"$0\")/dist/pnpm.mjs\" \"$@\"\n"
	if string(content) != wantWrapper {
		t.Errorf("bin/pnpm content = %q, want wrapper script %q", string(content), wantWrapper)
	}

	// Verify dist/pnpm.mjs was extracted from the tarball (present in all pnpm v11+ releases).
	mjsFP := filepath.Join(layer.Path, "bin/dist/pnpm.mjs")
	if _, err := os.Stat(mjsFP); err != nil {
		t.Errorf("os.Stat(%q) got error: %v, want nil", mjsFP, err)
	}
}

// mockTarballBytes returns a minimal .tar.gz that mirrors the flat structure of real pnpm v11+
// release tarballs: a "pnpm" native binary at root and a "dist/pnpm.mjs" JS entry point.
func mockTarballBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	pnpmBin := []byte("pnpm!")
	if err := tw.WriteHeader(&tar.Header{Name: "pnpm", Mode: 0755, Size: int64(len(pnpmBin))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(pnpmBin); err != nil {
		t.Fatal(err)
	}

	// dist/pnpm.mjs is present in every pnpm v11+ tarball and is referenced by the bash wrapper.
	mjsContent := []byte("// pnpm.mjs")
	if err := tw.WriteHeader(&tar.Header{Name: "dist/pnpm.mjs", Mode: 0644, Size: int64(len(mjsContent))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(mjsContent); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}
