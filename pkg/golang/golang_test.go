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

package golang

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/buildpacks/libcnb/v2"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestGoVersion(t *testing.T) {
	testCases := []struct {
		goVersion string
		want      string
	}{
		{
			goVersion: "go version go1.13 darwin/amd64",
			want:      "1.13",
		},
		{
			goVersion: "go version go1.14.7 darwin/amd64",
			want:      "1.14.7",
		},
		{
			goVersion: "go version go1.15beta2 darwin/amd64",
			want:      "1.15",
		},
		{
			goVersion: "go version go1.15rc1 darwin/amd64",
			want:      "1.15",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.goVersion, func(t *testing.T) {
			mockReadGoVersion(t, tc.goVersion)

			got, err := GoVersion(nil)
			if err != nil {
				t.Errorf("GoVersion(nil) failed unexpectedly; err=%s", err)
			}
			if got != tc.want {
				t.Errorf("GoVersion(nil) = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGoModVersion(t *testing.T) {
	testCases := []struct {
		name  string
		gomod string
		want  string
	}{
		{
			gomod: `
module dir

require (
    golang.org/x/textgo 0.3.0 // indirect
)`,
			want: "",
		},
		{
			gomod: `
module dir

go 1

require (
    golang.org/x/textgo 0.3.0 // indirect
)`,
			want: "",
		},
		{
			gomod: `
module dir

go 1.13

require (
    golang.org/x/textgo 0.3.0 // indirect
)`,
			want: "1.13",
		},
		{
			gomod: `
module dir

go 1.13.1

require (
    golang.org/x/textgo 0.3.0 // indirect
    rsc.io/quote v1.5.2
    rsc.io/quote/v3 v3.0.0
    rsc.io/sampler v1.3.1 // indirect
)`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

go 1.13.1
go 1.12.1

require (
    golang.org/x/textgo 0.3.0 // indirect
    rsc.io/quote v1.5.2
    rsc.io/quote/v3 v3.0.0
    rsc.io/sampler v1.3.1 // indirect
)`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

  go   1.13.1  

require (
    golang.org/x/textgo 0.3.0 // indirect
    rsc.io/quote v1.5.2
    rsc.io/quote/v3 v3.0.0
    rsc.io/sampler v1.3.1 // indirect
)`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

go1.13.1

require (
    golang.org/x/textgo 0.3.0 // indirect
    rsc.io/quote v1.5.2
    rsc.io/quote/v3 v3.0.0
    rsc.io/sampler v1.3.1 // indirect
)`,
			want: "",
		},
		{
			gomod: `
module dir

go 1.13
go 1.12
`,
			want: "1.13",
		},
		{
			gomod: `
module dir

go 1.13.1
go 1.12.1
`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

   go    1.13   
`,
			want: "1.13",
		},
		{
			gomod: `
module dir

   go    1.13.1   
`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

go 1.13.1
`,
			want: "1.13.1",
		},
		{
			gomod: `
module dir

go 1.13
`,
			want: "1.13",
		},
		{
			gomod: `
module dir

go 1.13.
`,
			want: "",
		},
		{
			gomod: `
module dir

go 1.
`,
			want: "",
		},
		{
			gomod: `
module dir

go 1
`,
			want: "",
		},
		{
			gomod: `
module dir

go .13.1
`,
			want: "",
		},
		{
			gomod: `
module dir

go .13.
`,
			want: "",
		},
		{
			gomod: `
module dir

go .13
`,
			want: "",
		},
		{
			gomod: `
module dir

go .
`,
			want: "",
		},
		{
			gomod: `
module dir

go 
`,
			want: "",
		},
		{
			gomod: `
module dir

go 1.1.1.1
`,
			want: "",
		},
		{
			gomod: `
module dir

1.13
`,
			want: "",
		},
	}

	for tci, tc := range testCases {
		t.Run(fmt.Sprintf("go.mod testcase %d", tci), func(t *testing.T) {
			dir, err := ioutil.TempDir("", tc.name)
			if err != nil {
				t.Fatalf("failing to create temp dir: %v", err)
			}
			defer os.RemoveAll(dir)

			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))

			if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte(tc.gomod), 0644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			got, err := GoModVersion(ctx)

			if err != nil {
				t.Fatalf("GoModVersion(%q) failed unexpectedly; err=%s", dir, err)
			}
			if got != tc.want {
				t.Errorf("GoModVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestSupportsAutoVendor(t *testing.T) {
	testCases := []struct {
		goVersion string
		goMod     string
		want      bool
	}{
		{
			goVersion: "go version go1.13 darwin/amd64",
			goMod:     "module dir\ngo 1.13",
			want:      false,
		},
		{
			goVersion: "go version go1.14 darwin/amd64",
			goMod:     "module dir\ngo 1.13",
			want:      false,
		},
		{
			goVersion: "go version go1.14 darwin/amd64",
			goMod:     "module dir\ngo 1.14",
			want:      true,
		},
		{
			goVersion: "go version go1.14.2 darwin/amd64",
			goMod:     "module v\ngo 1.14.1",
			want:      true,
		},
		{
			goVersion: "go version go1.15 darwin/amd64",
			goMod:     "module dir\ngo 1.15",
			want:      true,
		},
		{
			goVersion: "go version go1.13 darwin/amd64",
			goMod:     "module dir\ngo 1.14",
			want:      false,
		},
		{
			goMod: "",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.goMod, func(t *testing.T) {
			mockReadGoVersion(t, tc.goVersion)
			mockReadGoMod(t, tc.goMod)

			supported, err := SupportsAutoVendor(nil)

			if err != nil {
				t.Fatalf("VersionSupportsVendoredModules() failed unexpectedly; err=%s", err)
			}
			if supported != tc.want {
				t.Errorf("VersionSupportsVendoredModules() returned %v, wanted %v", supported, tc.want)
			}
		})
	}
}

func TestVersionMatches(t *testing.T) {
	testCases := []struct {
		goVersion    string
		goMod        string
		versionCheck string
		want         bool
	}{
		{
			goVersion:    "go version go1.13 darwin/amd64",
			goMod:        "module dir\ngo 1.13",
			versionCheck: ">1.13.0",
			want:         false,
		},
		{
			goVersion:    "go version go1.14 darwin/amd64",
			goMod:        "module dir\ngo 1.14",
			versionCheck: ">1.13.0",
			want:         true,
		},
		{
			goVersion:    "go version go1.15 darwin/amd64",
			goMod:        "module dir\ngo 1.15",
			versionCheck: ">=1.15.0",
			want:         true,
		},
		{
			goVersion:    "go version go1.15rc1 darwin/amd64",
			goMod:        "module dir\ngo 1.15",
			versionCheck: ">=1.15.0",
			want:         true,
		},
		{
			goVersion:    "go version go1.14.2 darwin/amd64",
			goMod:        "module v\ngo 1.14.1",
			versionCheck: ">=1.15.0",
			want:         false,
		},
		{
			goMod:        "",
			versionCheck: ">=1.15.0",
			want:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.goMod, func(t *testing.T) {
			mockReadGoVersion(t, tc.goVersion)
			mockReadGoMod(t, tc.goMod)

			supported, err := VersionMatches(nil, tc.versionCheck)

			if err != nil {
				t.Fatalf("VersionMatches() failed unexpectedly; err=%s", err)
			}
			if supported != tc.want {
				t.Errorf("VersionMatches() returned %v, wanted %v", supported, tc.want)
			}
		})
	}
}

func TestNewGoWorkspaceLayerHappyPath(t *testing.T) {
	testCases := []struct {
		Name            string
		ApplicationRoot string
		CacheEnabled    bool
		goMod           string
		goVersion       string
	}{
		{
			Name:            "go mod exists",
			ApplicationRoot: testdata.MustGetPath("testdata/gopath_layer/simple_gomod"),
			CacheEnabled:    true,
			goVersion:       "go version go1.14.2 darwin/amd64",
			goMod:           "module v\ngo 1.14.2",
		},
		{
			Name:            "go mod exists for go < 1.13",
			ApplicationRoot: testdata.MustGetPath("testdata/gopath_layer/simple_gomod"),
			CacheEnabled:    false,
			goVersion:       "go version go1.12.2 darwin/amd64",
			goMod:           "module v\ngo 1.12.1",
		},
		{
			Name:            "no go mod",
			ApplicationRoot: t.TempDir(),
			CacheEnabled:    false,
		},
	}

	mockCleanModCache(t)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			mockReadGoVersion(t, tc.goVersion)
			mockReadGoMod(t, tc.goMod)

			buildCtx := libcnb.BuildContext{
				Layers: libcnb.Layers{
					Path: t.TempDir(),
				},
			}
			ctx := gcp.NewContext(
				gcp.WithApplicationRoot(tc.ApplicationRoot),
				gcp.WithBuildContext(buildCtx))

			l, err := NewGoWorkspaceLayer(ctx)
			if err != nil {
				t.Fatalf("NewGoPathLayer() failed unexpectedly; err=%s", err)
			}
			if l.Cache != tc.CacheEnabled {
				t.Errorf("layer.Cache enablement mismatch: got %t, want %t", l.Cache, tc.CacheEnabled)
			}
			buildVars := map[string]string{
				"GOPATH":      l.Path,
				"GO111MODULE": "on",
				"GOPROXY":     "off",
			}
			for envVar, expectedVal := range buildVars {
				// libcnb appends an ".override" suffix to each env var
				val, ok := l.BuildEnvironment[fmt.Sprintf("%s.override", envVar)]
				if !ok {
					t.Fatalf("Layer missing required env var %v", envVar)
				}
				if val != expectedVal {
					t.Errorf("env var %q value mismatch: got %q, want %q", envVar, val, expectedVal)
				}
			}
		})
	}
}

func TestResolveGoVersion(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		want       string
		json       string
	}{
		{
			name: "all_stable",
			want: "1.16",
			json: `
[
 {
  "version": "go1.16",
  "stable": true
 },
 {
  "version": "go1.15.3",
  "stable": true
 },
 {
  "version": "go1.12.12",
  "stable": true
 }
]`,
		},
		{
			name: "recent_unstable",
			want: "1.15.3",
			json: `
[
 {
  "version": "go1.15.4",
  "stable": false
 },
 {
  "version": "go1.15.3",
  "stable": true
 },
 {
  "version": "go1.12.12",
  "stable": true
 }
]`,
		},
		{
			name:       "old exact major version",
			constraint: "1.12",
			want:       "1.12",
			json: `
[
 {
  "version": "go1.15.4",
  "stable": false
 },
 {
  "version": "go1.15.3",
  "stable": true
 }
]`,
		},
		{
			name:       "exact_unstable_rc_candidate",
			constraint: "1.21rc2",
			want:       "1.21rc2",
			json: `
[
{
"version": "go1.16",
"stable": true
},
{
"version": "go1.15.3",
"stable": true
},
{
"version": "go1.12.12",
"stable": true
}
]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithStatus(http.StatusOK),
				testserver.WithJSON(tc.json),
				testserver.WithMockURL(&goVersionsURL),
			)
			if v, err := ResolveGoVersion(tc.constraint); err != nil {
				t.Fatalf("resolveGoVersion(%q) failed: %v", tc.constraint, err)
			} else if v != tc.want {
				t.Errorf("resolveGoVersion(%q) = %q, want %q", tc.constraint, v, tc.want)
			}
		})
	}
}

// mockReadGoVersion mocks the readGoVersion
func mockReadGoVersion(t *testing.T, goVer string) {
	origReadGoVersion := readGoVersion
	readGoVersion = func(*gcp.Context) (string, error) { return goVer, nil }
	t.Cleanup(func() {
		readGoVersion = origReadGoVersion
	})
}

// mockReadGoMod mocks the readGoMod
func mockReadGoMod(t *testing.T, goMod string) {
	origReadGoMod := readGoMod
	readGoMod = func(*gcp.Context) (string, error) { return goMod, nil }
	t.Cleanup(func() {
		readGoMod = origReadGoMod
	})
}

// mockCleanModCache mocks the cleanModCache
func mockCleanModCache(t *testing.T) {
	origCleanModCache := cleanModCache
	cleanModCache = func(*gcp.Context) error { return nil }
	t.Cleanup(func() {
		cleanModCache = origCleanModCache
	})
}
