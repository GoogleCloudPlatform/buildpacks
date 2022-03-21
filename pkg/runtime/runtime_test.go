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

package runtime

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

func TestCheckOverride(t *testing.T) {
	testCases := []struct {
		name       string
		envRuntime string
		wantIn     bool
		wantOut    bool
	}{
		{
			name:       "with emptyu runtime returns nil",
			envRuntime: "",
		},
		{
			name:       "with runtime exact match opts in",
			envRuntime: "python",
			wantIn:     true,
		},
		{
			name:       "with runtime prefix match opts in",
			envRuntime: "python27",
			wantIn:     true,
		},
		{
			name:       "with runtime prefix mismatch opts out",
			envRuntime: "php55",
			wantOut:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envRuntime != "" {
				t.Setenv(env.Runtime, tc.envRuntime)
			}
			got := CheckOverride("python")
			if got == nil {
				if tc.wantIn || tc.wantOut {
					t.Errorf("CheckOverride(%q) envRuntime = (%v), got = (%v) want nil result",
						"python", tc.envRuntime, got)
				}
			} else if tc.wantIn && !got.Result().Pass {
				t.Errorf("CheckOverride(%q) envRuntime = (%v), got = (%v) want optOut result",
					"python", tc.envRuntime, got)
			} else if tc.wantOut && got.Result().Pass {
				t.Errorf("CheckOverride(%q) envRuntime = (%v), got = (%v) want optOut result",
					"python", tc.envRuntime, got)
			}
		})
	}
}
