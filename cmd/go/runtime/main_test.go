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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
)

func TestReadingVersion(t *testing.T) {
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
			ctx := gcp.NewContext(buildpack.Info{})

			dir, err := ioutil.TempDir("", tc.name)
			if err != nil {
				t.Fatalf("failing to create temp dir: %v", err)
			}
			defer os.RemoveAll(dir)

			if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte(tc.gomod), 0644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			if got := goModVersion(ctx, dir); got != tc.want {
				t.Errorf("goModVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestJSONVersionParse(t *testing.T) {
	testCases := []struct {
		name string
		want string
		json string
	}{
		{
			name: "all_stable",
			want: "1.13.3",
			json: `
[
 {
  "version": "go1.13.3",
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
			want: "1.12.12",
			json: `
[
 {
  "version": "go1.13.3",
  "stable": false
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
			if v, err := parseVersionJSON(tc.json); err != nil {
				t.Fatalf("parseVersionJSON() failed: %v", tc.name, err)
			} else if v != tc.want {
				t.Errorf("parseVersionJSON() = %q, want %q", tc.name, v, tc.want)
			}
		})
	}
}
