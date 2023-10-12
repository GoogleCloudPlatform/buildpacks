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
			name:       "java version resolver",
			constraint: "11.0",
			versions:   []string{"11.0.10+9", "11.0.11+9", "11.0.12+7", "11.0.13+8", "11.0.14+101", "11.0.14+9", "11.0.15+10", "11.0.16+101", "11.0.16+8", "11.0.17+8", "11.0.18+10", "11.0.19+7", "11.0.2+7", "11.0.2+9", "11.0.20+101", "11.0.20+8", "11.0.3+7", "11.0.4+11.1", "11.0.5+10.1", "11.0.6+10.1", "11.0.7+10.1", "11.0.8+10", "11.0.9+101", "11.0.9+11", "11.0.9+11.1", "16.0.0+36", "16.0.1+9", "16.0.2+7", "17.0.0+35", "17.0.1+12", "17.0.2+8", "17.0.3+7", "17.0.4+101", "17.0.4+8", "17.0.5+8", "17.0.6+10", "17.0.7+7", "17.0.8+101", "17.0.8+7", "18.0.0+36", "18.0.1+10", "18.0.2+101", "18.0.2+9", "19.0.0+36", "19.0.1+10", "19.0.2+7", "8.0.192+12", "8.0.202+8", "8.0.212+3", "8.0.212+4", "8.0.222+10.1", "8.0.232+9.1", "8.0.242+8.1", "8.0.252+9.1", "8.0.262+10", "8.0.265+1", "8.0.272+10", "8.0.275+1", "8.0.282+8", "8.0.292+10", "8.0.302+8", "8.0.312+7", "8.0.322+6", "8.0.332+9", "8.0.342+7", "8.0.345+1", "8.0.352+8", "8.0.362+9", "8.0.372+7", "8.0.382+5"},
			want:       "11.0.20+8",
		},
		{
			name:       "java version lts resolver",
			constraint: "21.0",
			versions:   []string{"11.0.10+9", "11.0.11+9", "11.0.12+7", "11.0.13+8", "11.0.14+101", "11.0.14+9", "11.0.15+10", "11.0.16+101", "11.0.16+8", "11.0.17+8", "11.0.18+10", "11.0.19+7", "11.0.2+7", "11.0.2+9", "11.0.20+101", "11.0.20+8", "11.0.3+7", "11.0.4+11.1", "11.0.5+10.1", "11.0.6+10.1", "11.0.7+10.1", "11.0.8+10", "11.0.9+101", "11.0.9+11", "11.0.9+11.1", "16.0.0+36", "16.0.1+9", "16.0.2+7", "17.0.0+35", "17.0.1+12", "17.0.2+8", "17.0.3+7", "17.0.4+101", "17.0.4+8", "17.0.5+8", "17.0.6+10", "17.0.7+7", "17.0.8+101", "17.0.8+7", "18.0.0+36", "18.0.1+10", "18.0.2+101", "18.0.2+9", "19.0.0+36", "19.0.1+10", "19.0.2+7", "8.0.192+12", "8.0.202+8", "8.0.212+3", "8.0.212+4", "8.0.222+10.1", "8.0.232+9.1", "8.0.242+8.1", "8.0.252+9.1", "8.0.262+10", "8.0.265+1", "8.0.272+10", "8.0.275+1", "8.0.282+8", "8.0.292+10", "8.0.302+8", "8.0.312+7", "8.0.322+6", "8.0.332+9", "8.0.342+7", "8.0.345+1", "8.0.352+8", "8.0.362+9", "8.0.372+7", "8.0.382+5", "21.0.0+35.0.LTS"},
			want:       "21.0.0+35.0.LTS",
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
