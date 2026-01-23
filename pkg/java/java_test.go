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

package java

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

func TestFindManifestValueFromJar(t *testing.T) {
	testCases := []struct {
		name             string
		key              string
		manifestContents string
		want             string
	}{
		{
			name: "simple case",
			key:  "Main-Class",
			manifestContents: `Main-Class: test
another: example`,
			want: "test",
		},
		{
			name: "with line continuation",
			key:  "Start-Class",
			// The manifest spec states that a continuation line must end with a trailing space.
			manifestContents: `simple: example 
 wrapping
Start-Class: example`,
			want: "example",
		},
		{
			name: "main-class with line continuation",
			key:  "Main-Class",
			manifestContents: `simple: example 
 wrapping
Main-Class: example 
 line wrap
New-Entry: example`,
			want: "example",
		},
		{
			name: "no start class entry",
			key:  "Start-Class",
			manifestContents: `Not-Start-Class: test
another: example`,
			want: "",
		},
		{
			name: "main class with preceding space",
			key:  "Main-Class",
			manifestContents: `simple: example
 wrapping
 Main-Class: example`,
			want: "",
		},
		{
			name:             "start class with no entry",
			key:              "Start-Class",
			manifestContents: `Start-Class: `,
			want:             "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jarPath := setupTestJar(t, []byte(tc.manifestContents))
			got, err := FindManifestValueFromJar(jarPath, tc.key)
			if err != nil {
				t.Errorf("FindManifestValueFromJar() errored: %v", err)
			}

			if got != tc.want {
				t.Errorf("FindManifestValueFromJar()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestMainFromManifest(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
		want             string
	}{
		{
			name:             "simple case",
			manifestContents: `Main-Class: test`,
			want:             "test",
		},
		{
			name: "2 line case",
			manifestContents: `ex: example
Main-Class: test`,
			want: "test",
		},
		{
			name: "3 line case",
			manifestContents: `ex: example
Main-Class: test
another: example`,
			want: "test",
		},
		{
			name: "3 line with trailing space case",
			manifestContents: `ex: example 
Main-Class: test 
another: example`,
			want: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mfPath := setupTestManifest(t, []byte(tc.manifestContents))
			ctx := gcp.NewContext()
			got, err := MainFromManifest(ctx, mfPath)
			if err != nil {
				t.Errorf("MainFromMainfest() errored: %v", err)
			}

			if got != tc.want {
				t.Errorf("MainFromMainfest()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestMainFromManifestFail(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name:             "empty manifest",
			manifestContents: ``,
		},
		{
			name:             "no main-class entry found",
			manifestContents: `key: value`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mfPath := setupTestManifest(t, []byte(tc.manifestContents))
			ctx := gcp.NewContext()
			_, err := MainFromManifest(ctx, mfPath)
			if err == nil {
				t.Error("MainFromMainfest() did not error as expected")
			}
		})
	}
}

func TestCheckCacheNewDateMiss(t *testing.T) {
	testCases := []struct {
		name            string
		expiryTimestamp string
	}{
		{
			name:            "empty string",
			expiryTimestamp: "",
		},
		{
			name:            "invalid format",
			expiryTimestamp: "invalid format",
		},
		{
			name:            "old expiry date",
			expiryTimestamp: time.Now().Truncate(repoExpiration).Format(dateFormat),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()

			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)
			ctx.SetMetadata(m2CachedRepo, "expiry_timestamp", tc.expiryTimestamp)

			if err := CheckCacheExpiration(ctx, m2CachedRepo); err != nil {
				t.Fatalf("CheckCacheExpiration() unexpected error = %q", err.Error())
			}
			metaExpiry := ctx.GetMetadata(m2CachedRepo, "expiry_timestamp")
			if metaExpiry == tc.expiryTimestamp {
				t.Errorf("checkCacheExpiration() did not set new date when expected to with ExpiryTimestamp: %q", metaExpiry)
			}
			testFilePathExists, err := ctx.FileExists(testFilePath)
			if err != nil {
				t.Fatalf("Error checking if file exists: %v", err)
			}
			if testFilePathExists {
				if err := os.RemoveAll(testFilePath); err != nil {
					t.Errorf("error cleaning up: %v", err)
				}
				t.Errorf("checkCacheExpiration() did not clear layer")
			}
		})
	}
}

func TestCheckCacheNewDateHit(t *testing.T) {
	testCases := []struct {
		name            string
		expiryTimestamp string
	}{
		{
			name:            "expiry date in the future",
			expiryTimestamp: time.Now().Add(repoExpiration).Format(dateFormat),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			testFilePath, m2CachedRepo := setupTestLayer(t, ctx)
			ctx.SetMetadata(m2CachedRepo, "expiry_timestamp", tc.expiryTimestamp)

			if err := CheckCacheExpiration(ctx, m2CachedRepo); err != nil {
				t.Fatalf("CheckCacheExpiration() unexpected error = %q", err.Error())
			}
			if got, want := ctx.GetMetadata(m2CachedRepo, "expiry_timestamp"), tc.expiryTimestamp; got != want {
				t.Errorf("checkCacheExpiration() set new date when expected not to with ExpiryTimestamp: %q", got)
			}
			if _, err := os.Stat(testFilePath); err != nil {
				t.Errorf("checkCacheExpiration() cleared layer")
			}
			// Clean up layer for next test case.
			if err := os.RemoveAll(testFilePath); err != nil {
				t.Fatalf("error cleaning up: %v", err)
			}
		})
	}
}

func setupTestLayer(t *testing.T, ctx *gcp.Context) (string, *libcnb.Layer) {
	t.Helper()
	testLayerRoot := t.TempDir()
	testFilePath := filepath.Join(testLayerRoot, "testfile")
	_, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("error creating %s: %v", testFilePath, err)
	}
	m2CachedRepo := &libcnb.Layer{
		Path:     testLayerRoot,
		Metadata: map[string]interface{}{},
	}
	return testFilePath, m2CachedRepo
}

func setupTestManifest(t *testing.T, mfContent []byte) string {
	t.Helper()
	mfPath := filepath.Join(t.TempDir(), "TEST.MF")
	if err := ioutil.WriteFile(mfPath, mfContent, 0644); err != nil {
		t.Fatalf("writing to file %s: %v", mfPath, err)
	}
	return mfPath
}

func TestExecutableJar(t *testing.T) {
	execManifest := []byte("Main-Class: com.example.Main")
	plainManifest := []byte("Color: blue")
	testCases := []struct {
		name      string
		buildable string
		jars      map[string][]byte // map[rel_path]manifest_content
		want      string
		wantErr   bool
	}{
		{
			name:    "no_jars",
			wantErr: true,
		},
		{
			name: "one_exec_jar_target",
			jars: map[string][]byte{
				"target/exec.jar": execManifest,
			},
			want: "target/exec.jar",
		},
		{
			name: "one_exec_jar_build_libs",
			jars: map[string][]byte{
				"build/libs/exec.jar": execManifest,
			},
			want: "build/libs/exec.jar",
		},
		{
			name: "two_exec_jar_target",
			jars: map[string][]byte{
				"target/exec1.jar": execManifest,
				"target/exec2.jar": execManifest,
			},
			wantErr: true,
		},
		{
			name:      "buildable_jar_in_buildable_target",
			buildable: "app",
			jars: map[string][]byte{
				"app/target/exec.jar": execManifest,
			},
			want: "app/target/exec.jar",
		},
		{
			name:      "buildable_jar_in_buildable",
			buildable: "app",
			jars: map[string][]byte{
				"app/exec.jar": execManifest,
			},
			want: "app/exec.jar",
		},
		{
			name:      "buildable_jar_in_target",
			buildable: "app",
			jars: map[string][]byte{
				"target/exec.jar": execManifest,
			},
			want: "target/exec.jar",
		},
		{
			name: "multiple_jars_one_exec",
			jars: map[string][]byte{
				"target/a.jar": plainManifest,
				"target/b.jar": execManifest,
			},
			want: "target/b.jar",
		},
		{
			name:      "buildable_multiple_jars_one_exec",
			buildable: "app",
			jars: map[string][]byte{
				"app/target/a.jar": plainManifest,
				"app/target/b.jar": execManifest,
			},
			want: "app/target/b.jar",
		},
		{
			name:      "buildable_and_root_exec_jars",
			buildable: "app",
			jars: map[string][]byte{
				"app/target/exec1.jar": execManifest,
				"target/exec2.jar":     execManifest,
			},
			want: "app/target/exec1.jar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			for path, manifest := range tc.jars {
				fullPath := filepath.Join(appDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("Failed to create dir for jar: %v", err)
				}
				if err := ioutil.WriteFile(fullPath, jarContent(t, manifest), 0644); err != nil {
					t.Fatalf("Failed to write jar file: %v", err)
				}
			}

			if tc.buildable != "" {
				t.Setenv(env.Buildable, tc.buildable)
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			got, err := ExecutableJar(ctx)

			if tc.wantErr && err == nil {
				t.Errorf("ExecutableJar() succeeded, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ExecutableJar() failed with %v, want success", err)
			}

			if !tc.wantErr && err == nil {
				want := filepath.Join(appDir, tc.want)
				if got != want {
					t.Errorf("ExecutableJar()=%q, want %q", got, want)
				}
			}
		})
	}
}

func jarContent(t *testing.T, mfContent []byte) []byte {
	t.Helper()
	var buff bytes.Buffer
	w := zip.NewWriter(&buff)
	defer w.Close()
	f, err := w.Create(filepath.Join("META-INF", "MANIFEST.MF"))
	if err != nil {
		t.Fatalf("creating zip entry: %v", err)
	}
	for i := 0; i < len(mfContent); {
		n, err := f.Write(mfContent)
		if err != nil {
			t.Fatalf("writing bytes: %v", err)
		}
		i += n
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	return buff.Bytes()
}

func setupTestJar(t *testing.T, mfContent []byte) string {
	t.Helper()
	jarPath := filepath.Join(t.TempDir(), "test.jar")
	if err := ioutil.WriteFile(jarPath, jarContent(t, mfContent), 0644); err != nil {
		t.Fatalf("writing to file %s: %v", jarPath, err)
	}
	return jarPath
}
