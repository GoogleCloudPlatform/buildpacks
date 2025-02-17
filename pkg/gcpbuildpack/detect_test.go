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

package gcpbuildpack

import (
	"os"
	"reflect"
	"testing"

	"github.com/buildpacks/libcnb/v2"
)

func TestOpt(t *testing.T) {
	testCases := []struct {
		name   string
		pass   bool
		reason string
		opts   []DetectResultOption
		want   libcnb.DetectResult
	}{
		{
			name:   "reason only pass",
			pass:   true,
			reason: "some reason",
			want: libcnb.DetectResult{
				Pass: true,
			},
		},
		{
			name:   "reason only fail",
			pass:   false,
			reason: "some reason",
			want: libcnb.DetectResult{
				Pass: false,
			},
		},
		{
			name:   "with build plan",
			pass:   true,
			reason: "some reason",
			opts: []DetectResultOption{
				WithBuildPlans(libcnb.BuildPlan{
					Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}},
					Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}},
				}),
			},
			want: libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{{
					Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}},
					Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}},
				}},
			},
		},
		{
			name:   "with multiple build plans",
			pass:   true,
			reason: "some reason",
			opts: []DetectResultOption{
				WithBuildPlans(
					libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
					libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
				),
			},
			want: libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
					libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := opt(tc.pass, tc.reason, tc.opts...)
			if want, got := tc.reason, result.Reason(); want != got {
				t.Errorf("result.Reason() = %s, want %s", got, want)
			}
			if want, got := tc.want, result.Result(); !reflect.DeepEqual(want, got) {
				t.Errorf("result.Result() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestOptIn(t *testing.T) {
	result := OptIn("some reason", WithBuildPlans(
		libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
		libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
	))

	if want, got := "Opting in: some reason", result.Reason(); want != got {
		t.Errorf("result.Reason() = %s, want %s", got, want)
	}

	want := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
			libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
		},
	}

	if got := result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf("result.Result() = %#v, want %#v", got, want)
	}
}

func TestOptOut(t *testing.T) {
	result := OptOut("some reason", WithBuildPlans(
		libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
		libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
	))

	if want, got := "Opting out: some reason", result.Reason(); want != got {
		t.Errorf("result.Reason() = %s, want %s", got, want)
	}

	want := libcnb.DetectResult{
		Pass: false,
		Plans: []libcnb.BuildPlan{
			libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
			libcnb.BuildPlan{Requires: []libcnb.BuildPlanRequire{{Name: "some-require"}}},
		},
	}

	if got := result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf("result.Result() = %#v, want %#v", got, want)
	}
}

func TestOptInVariants(t *testing.T) {
	opt := WithBuildPlans(libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}})

	wantResult := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
		},
	}

	// OptInFileFound
	result := OptInFileFound("my-file", opt)
	if want, got := "Opting in: found my-file", result.Reason(); want != got {
		t.Errorf(`OptInFileFound("my-file", opt).Reason() = %s, want %s`, got, want)
	}
	if want, got := wantResult, result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf(`OptInFileFound("my-file", opt).Result() = %#v, want %#v`, got, want)
	}

	// OptInEnvSet
	if err := os.Setenv("MY_ENV", "MY_VAL"); err != nil {
		t.Fatalf("Setting MY_ENV env var: %v", err)
	}
	defer os.Unsetenv("MY_ENV")

	result = OptInEnvSet("MY_ENV", opt)
	if want, got := `Opting in: MY_ENV set to "MY_VAL"`, result.Reason(); want != got {
		t.Errorf(`OptInEnvSet("MY_ENV", opt).Reason() = %s, want %s`, got, want)
	}
	if want, got := wantResult, result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf(`OptInEnvSet("MY_ENV", opt).Result() = %#v, want %#v`, got, want)
	}
}

func TestOptOutVariants(t *testing.T) {
	opt := WithBuildPlans(libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}})

	wantResult := libcnb.DetectResult{
		Pass: false,
		Plans: []libcnb.BuildPlan{
			libcnb.BuildPlan{Provides: []libcnb.BuildPlanProvide{{Name: "some-provide"}}},
		},
	}

	// OptOutFileNotFound
	result := OptOutFileNotFound("my-file", opt)
	if want, got := "Opting out: my-file not found", result.Reason(); want != got {
		t.Errorf(`OptOutFileNotFound("my-file", opt).Reason() = %s, want %s`, got, want)
	}
	if want, got := wantResult, result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf(`OptOutFileNotFound("my-file", opt).Result() = %#v, want %#v`, got, want)
	}

	// OptOutEnvNotSet
	result = OptOutEnvNotSet("MY_ENV", opt)
	if want, got := `Opting out: MY_ENV not set`, result.Reason(); want != got {
		t.Errorf(`OptOutEnvNotSet("MY_ENV", opt).Reason() = %s, want %s`, got, want)
	}
	if want, got := wantResult, result.Result(); !reflect.DeepEqual(want, got) {
		t.Errorf(`OptOutEnvNotSet("MY_ENV", opt).Result() = %#v, want %#v`, got, want)
	}
}
