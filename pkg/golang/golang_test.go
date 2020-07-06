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
	"os"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
)

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

			ctx := gcp.NewContextForTests(buildpack.Info{}, dir)

			if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte(tc.gomod), 0644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			if got := GoModVersion(ctx); got != tc.want {
				t.Errorf("GoModVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestSupportsNoGoMod(t *testing.T) {
	testCases := []struct {
		goVersion string
		want      bool
	}{
		{
			goVersion: "go version go1.11 darwin/amd64",
			want:      true,
		},
		{
			goVersion: "go version go1.11.1 darwin/amd64",
			want:      true,
		},
		{
			goVersion: "go version go1.13 darwin/amd64",
			want:      true,
		},
		{
			goVersion: "go version go1.13.3 darwin/amd64",
			want:      true,
		},
		{
			goVersion: "go version go1.10 darwin/amd64",
			want:      true,
		},
		{
			goVersion: "go version go1.14 darwin/amd64",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.goVersion, func(t *testing.T) {
			defer func(fn func(*gcp.Context) string) { readGoVersion = fn }(readGoVersion)
			readGoVersion = func(*gcp.Context) string { return tc.goVersion }

			supported := SupportsNoGoMod(nil)

			if supported != tc.want {
				t.Errorf("VersionSupportsNoGoModFile() returned %v, wanted %v", supported, tc.want)
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
			defer func(fn func(*gcp.Context) string) { readGoVersion = fn }(readGoVersion)
			readGoVersion = func(*gcp.Context) string { return tc.goVersion }

			defer func(fn func(*gcp.Context) string) { readGoMod = fn }(readGoMod)
			readGoMod = func(*gcp.Context) string { return tc.goMod }

			supported := SupportsAutoVendor(nil)

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
			defer func(fn func(*gcp.Context) string) { readGoVersion = fn }(readGoVersion)
			readGoVersion = func(*gcp.Context) string { return tc.goVersion }

			defer func(fn func(*gcp.Context) string) { readGoMod = fn }(readGoMod)
			readGoMod = func(*gcp.Context) string { return tc.goMod }

			supported := VersionMatches(nil, tc.versionCheck)

			if supported != tc.want {
				t.Errorf("VersionMatches() returned %v, wanted %v", supported, tc.want)
			}
		})
	}
}
