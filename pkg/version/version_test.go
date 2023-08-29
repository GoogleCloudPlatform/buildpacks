// Copyright 2022 Google LLC
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

package version

import (
	"testing"
)

func TestResolveVersion(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		opts       []ResolveVersionOption
		versions   []string
		want       string
		wantError  bool
	}{
		{
			name:       "tilda specifier",
			constraint: "~1.2.1",
			versions:   []string{"1.2.3", "1.2.4", "1.3.0", "0.1.2", "2.0.0"},
			want:       "1.2.4",
		},
		{
			name:       "carat specifier",
			constraint: "^1.2.1",
			versions:   []string{"1.2.3", "1.2.4", "1.3.0", "0.1.2", "2.0.0"},
			want:       "1.3.0",
		},
		{
			name:       "complex constraint",
			constraint: ">= 1.2.3, < 1.2.4",
			versions:   []string{"1.2.3", "1.2.4", "1.3.0", "0.1.2", "2.0.0"},
			want:       "1.2.3",
		},
		{
			name:     "default to newest",
			versions: []string{"1.2.3", "1.2.4", "1.3.0", "0.1.2", "2.0.0"},
			want:     "2.0.0",
		},
		{
			name:       "strips prefix",
			constraint: "v10.1.1",
			versions:   []string{"v10.1.1"},
			want:       "10.1.1",
		},
		{
			name:       "no santization prefix",
			opts:       []ResolveVersionOption{WithoutSanitization},
			constraint: "v10.1.1",
			versions:   []string{"v10.1.1"},
			want:       "v10.1.1",
		},
		{
			name:       "no santization zero padding",
			opts:       []ResolveVersionOption{WithoutSanitization},
			constraint: "*",
			versions:   []string{"1.16"},
			want:       "1.16",
		},
		{
			name:       "invalid constraint",
			constraint: "xyz",
			versions:   []string{"v10.1.1"},
			wantError:  true,
		},
		{
			name:       "no matching version",
			constraint: ">=2.0.0",
			versions:   []string{"1.2.3", "1.2.4"},
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveVersion(tc.constraint, tc.versions, tc.opts...)
			if tc.wantError != (err != nil) {
				t.Errorf("ResolveVersion(%q, %v) got error: %v, want error?: %v", tc.constraint, tc.versions, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf("ResolveVersion(%q, %v) = %q, want %q", tc.constraint, tc.versions, got, tc.want)
			}
		})
	}
}

func TestIsExactSemver(t *testing.T) {
	testCases := []struct {
		version string
		want    bool
	}{
		{
			version: "v10.1.1",
			want:    true,
		},
		{
			version: "1.1",
			want:    false,
		},
		{
			version: "2",
			want:    false,
		},
		{
			version: "",
			want:    false,
		},
		{
			version: "1.x.x",
			want:    false,
		},
		{
			version: "~1.0.0",
			want:    false,
		},
		{
			version: ">=1.0.0",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			got := IsExactSemver(tc.version)
			if got != tc.want {
				t.Errorf("IsExactSemver(%q) = %t, want %t", tc.version, got, tc.want)
			}
		})
	}
}

func TestIsReleaseCandidate(t *testing.T) {
	testCases := []struct {
		version string
		want    bool
	}{
		{
			version: "3.12.0rc1",
			want:    true,
		},
		{
			version: "3.12.0",
			want:    false,
		},
		{
			version: "3.12",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			got := IsReleaseCandidate(tc.version)
			if got != tc.want {
				t.Errorf("IsReleaseCandidate(%q) = %t, want %t", tc.version, got, tc.want)
			}
		})
	}
}
