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

package runtime

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
	"github.com/buildpacks/libcnb/v2"
)

const testRuntimeVersions = `{"nodejs24":"24.22.1","python314":"3.14.10","go126":"1.26.8","go127":"1.27.3","static24":"24"}`

func TestSelectAutodetectRuntime(t *testing.T) {
	testCases := []struct {
		name       string
		versions   string
		language   string
		constraint string
		want       *AutodetectedRuntime
	}{
		{
			name:     "env not set",
			language: "nodejs",
		},
		{
			name:       "nodejs stack default",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.*.*",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs open range picks eligible runtime",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: ">=16.0.0",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs caret range",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "^24",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs x range",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.x",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs version not eligible",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "22.x",
		},
		{
			name:       "nodejs exact version matches",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.22.1",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs pinned version falls back to release line",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.21.0",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs pinned version newer than eligible version",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.23.0",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs pinned minor version falls back to release line",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "24.21",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs pinned version with v prefix",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "v24.21.0",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
		{
			name:       "nodejs pinned version of other release line",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "22.1.0",
		},
		{
			name:       "nodejs range does not fall back",
			versions:   testRuntimeVersions,
			language:   "nodejs",
			constraint: "~24.21.0",
		},
		{
			name:       "python pinned version falls back to release line",
			versions:   testRuntimeVersions,
			language:   "python",
			constraint: "3.14.9",
			want:       &AutodetectedRuntime{Name: "python314", Version: "3.14.10"},
		},
		{
			name:       "python pinned version does not match a longer minor version",
			versions:   `{"python314":"3.14.10"}`,
			language:   "python",
			constraint: "3.140.1",
		},
		{
			name:       "go pinned version falls back to release line",
			versions:   testRuntimeVersions,
			language:   "go",
			constraint: "1.26.1",
			want:       &AutodetectedRuntime{Name: "go126", Version: "1.26.8"},
		},
		{
			name:       "python minor version",
			versions:   testRuntimeVersions,
			language:   "python",
			constraint: "3.14",
			want:       &AutodetectedRuntime{Name: "python314", Version: "3.14.10"},
		},
		{
			name:       "python stack default",
			versions:   testRuntimeVersions,
			language:   "python",
			constraint: "3.14.*",
			want:       &AutodetectedRuntime{Name: "python314", Version: "3.14.10"},
		},
		{
			name:       "python version not eligible",
			versions:   testRuntimeVersions,
			language:   "python",
			constraint: "3.12",
		},
		{
			name:       "go picks matching line",
			versions:   testRuntimeVersions,
			language:   "go",
			constraint: "1.26.*",
			want:       &AutodetectedRuntime{Name: "go126", Version: "1.26.8"},
		},
		{
			name:       "go picks highest matching runtime",
			versions:   testRuntimeVersions,
			language:   "go",
			constraint: ">=1.26",
			want:       &AutodetectedRuntime{Name: "go127", Version: "1.27.3"},
		},
		{
			name:     "static without constraint",
			versions: testRuntimeVersions,
			language: "static",
			want:     &AutodetectedRuntime{Name: "static24", Version: "24"},
		},
		{
			name:       "language not eligible",
			versions:   testRuntimeVersions,
			language:   "ruby",
			constraint: "*",
		},
		{
			name:       "language prefix does not match other runtimes",
			versions:   `{"golang1":"1.0.0"}`,
			language:   "go",
			constraint: "*",
		},
		{
			name:       "surrounding whitespace is ignored",
			versions:   ` {"nodejs24": "24.22.1"} `,
			language:   "nodejs",
			constraint: "24",
			want:       &AutodetectedRuntime{Name: "nodejs24", Version: "24.22.1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.XGoogleRuntimeVersions, tc.versions)

			got, err := SelectAutodetectRuntime(tc.language, tc.constraint)
			if err != nil {
				t.Fatalf("SelectAutodetectRuntime(%q, %q) failed: %v", tc.language, tc.constraint, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SelectAutodetectRuntime(%q, %q) returned unexpected diff (-want +got):\n%s", tc.language, tc.constraint, diff)
			}
		})
	}
}

func TestSelectAutodetectRuntimeErrors(t *testing.T) {
	testCases := []struct {
		name     string
		versions string
	}{
		{
			name:     "not json",
			versions: "nodejs24=24.22.1",
		},
		{
			name:     "not an object",
			versions: `["nodejs24"]`,
		},
		{
			name:     "non-string version",
			versions: `{"nodejs24":24}`,
		},
		{
			name:     "empty version",
			versions: `{"nodejs24":""}`,
		},
		{
			name:     "empty runtime",
			versions: `{"":"24.22.1"}`,
		},
		{
			name:     "duplicate version",
			versions: `{"nodejs24":"24.22.1","nodejs25":"24.22.1"}`,
		},
		{
			name:     "version not semver",
			versions: `{"nodejs24":"latest"}`,
		},
		{
			name: "invalid version of another language rejects all",
			// One bad entry is a platform bug, so no runtime is selected for any language.
			versions: `{"nodejs24":"24.22.1","python314":"3.14.x.1"}`,
		},
		{
			name:     "release candidate version",
			versions: `{"nodejs24":"24.22.1","python315":"3.15.0rc3"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.XGoogleRuntimeVersions, tc.versions)

			if got, err := SelectAutodetectRuntime("nodejs", "*"); err == nil {
				t.Errorf("SelectAutodetectRuntime(%q, %q) = %v, want error", "nodejs", "*", got)
			}
		})
	}
}

func TestSelectAutodetectRuntimeInvalidConstraint(t *testing.T) {
	t.Setenv(env.XGoogleRuntimeVersions, testRuntimeVersions)

	for _, constraint := range []string{">=>24", "24.x.y.z", "not a version"} {
		if got, err := SelectAutodetectRuntime("nodejs", constraint); err == nil {
			t.Errorf("SelectAutodetectRuntime(%q, %q) = %v, want error", "nodejs", constraint, got)
		}
	}
}

func TestWithAutodetectPlan(t *testing.T) {
	constraintErr := errors.New("bad version file")
	testCases := []struct {
		name          string
		versions      string
		constraint    string
		constraintErr error
		wantPlans     []libcnb.BuildPlan
	}{
		{
			name:       "env not set",
			constraint: "24",
		},
		{
			name:       "eligible runtime",
			versions:   testRuntimeVersions,
			constraint: "24",
			wantPlans: []libcnb.BuildPlan{{
				Provides: []libcnb.BuildPlanProvide{{Name: "google.runtime"}},
				Requires: []libcnb.BuildPlanRequire{{
					Name:     "google.runtime",
					Metadata: map[string]any{"runtime": "nodejs24", "version": "24.22.1"},
				}},
			}},
		},
		{
			name:       "no eligible runtime",
			versions:   testRuntimeVersions,
			constraint: "22",
		},
		{
			name:          "constraint error",
			versions:      testRuntimeVersions,
			constraintErr: constraintErr,
		},
		{
			name:       "invalid env",
			versions:   "nodejs24",
			constraint: "24",
		},
		{
			name:       "invalid version in env",
			versions:   `{"nodejs24":"latest"}`,
			constraint: "24",
		},
		{
			name:       "invalid constraint",
			versions:   testRuntimeVersions,
			constraint: ">=>24",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.XGoogleRuntimeVersions, tc.versions)
			called := false
			constraint := func() (string, error) {
				called = true
				return tc.constraint, tc.constraintErr
			}

			opt := WithAutodetectPlan(gcp.NewContext(), "nodejs", constraint)
			got := gcp.OptIn("test", opt).Result().Plans

			if tc.versions == "" && called {
				t.Error("WithAutodetectPlan() called constraint when the env var is not set")
			}
			if diff := cmp.Diff(tc.wantPlans, got); diff != "" {
				t.Errorf("WithAutodetectPlan() plans returned unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWithAutodetectPlanIsLazy(t *testing.T) {
	t.Setenv(env.XGoogleRuntimeVersions, testRuntimeVersions)
	called := false

	WithAutodetectPlan(gcp.NewContext(), "nodejs", func() (string, error) {
		called = true
		return "24", nil
	})

	if called {
		t.Error("WithAutodetectPlan() called constraint before the option was applied")
	}
}

func TestCheckOverrideAppliesOptionsToOptIn(t *testing.T) {
	plan := libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "test"}}}
	t.Setenv(env.Runtime, "nodejs24")

	result := CheckOverride("nodejs", gcp.WithBuildPlans(plan))

	if result == nil || !result.Result().Pass {
		t.Fatalf("CheckOverride(%q) = %v, want an opt-in result", "nodejs", result)
	}
	if diff := cmp.Diff([]libcnb.BuildPlan{plan}, result.Result().Plans); diff != "" {
		t.Errorf("CheckOverride(%q) plans returned unexpected diff (-want +got):\n%s", "nodejs", diff)
	}
}
