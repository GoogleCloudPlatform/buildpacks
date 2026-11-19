// Copyright 2026 Google LLC
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

package autodetect

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseRuntimeVersions(t *testing.T) {
	testCases := []struct {
		name    string
		raw     string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "empty",
			raw:  " ",
		},
		{
			name: "valid",
			raw:  `{"nodejs24":"24.22.1","python314":"3.14.10","static24":"24"}`,
			want: map[string]string{"nodejs24": "24.22.1", "python314": "3.14.10", "static24": "24"},
		},
		{
			name:    "not json",
			raw:     "nodejs24=24.22.1",
			wantErr: true,
		},
		{
			name:    "empty name",
			raw:     `{"":"24.22.1"}`,
			wantErr: true,
		},
		{
			name:    "empty version",
			raw:     `{"nodejs24":""}`,
			wantErr: true,
		},
		{
			name:    "not semver",
			raw:     `{"nodejs24":"latest"}`,
			wantErr: true,
		},
		{
			name:    "release candidate",
			raw:     `{"python315":"3.15.0rc3"}`,
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRuntimeVersions(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseRuntimeVersions(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseRuntimeVersions(%q) returned unexpected diff (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

const nodePlan = `
[[entries]]
  [[entries.providers]]
    id = "google.nodejs.runtime"
  [[entries.requires]]
    name = "google.runtime"
    [entries.requires.metadata]
      runtime = "nodejs24"
      version = "24.22.1"
`

func TestRuntimeFromPlan(t *testing.T) {
	runtimes := map[string]string{"nodejs24": "24.22.1", "python314": "3.14.10"}
	testCases := []struct {
		name    string
		plan    string
		want    *Runtime
		wantErr bool
	}{
		{
			name: "detected",
			plan: nodePlan,
			want: &Runtime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name: "other entries are ignored",
			plan: `
[[entries]]
  [[entries.providers]]
    id = "google.nodejs.npm"
  [[entries.requires]]
    name = "node"
` + nodePlan,
			want: &Runtime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name: "same runtime recorded twice",
			plan: nodePlan + nodePlan,
			want: &Runtime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name: "no runtime recorded",
			plan: `
[[entries]]
  [[entries.requires]]
    name = "node"
`,
		},
		{
			name: "empty plan",
		},
		{
			name: "different runtimes recorded",
			plan: nodePlan + `
[[entries]]
  [[entries.requires]]
    name = "google.runtime"
    [entries.requires.metadata]
      runtime = "python314"
      version = "3.14.10"
`,
			wantErr: true,
		},
		{
			name: "version may not be detected",
			plan: `
[[entries]]
  [[entries.requires]]
    name = "google.runtime"
    [entries.requires.metadata]
      runtime = "nodejs24"
      version = "24.21.0"
`,
			wantErr: true,
		},
		{
			name: "runtime may not be detected",
			plan: `
[[entries]]
  [[entries.requires]]
    name = "google.runtime"
    [entries.requires.metadata]
      runtime = "nodejs22"
      version = "22.20.0"
`,
			wantErr: true,
		},
		{
			name: "no metadata",
			plan: `
[[entries]]
  [[entries.requires]]
    name = "google.runtime"
`,
			wantErr: true,
		},
		{
			name:    "not toml",
			plan:    "not toml [",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RuntimeFromPlan([]byte(tc.plan), runtimes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("RuntimeFromPlan() for %q error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("RuntimeFromPlan() for %q returned unexpected diff (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}
