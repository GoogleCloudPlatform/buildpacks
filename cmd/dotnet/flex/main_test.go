// Copyright 2023 Google LLC
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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

const (
	defaultStack = "com.stack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name string
		env  []string
		want int
	}{
		{
			name: "flex envar set",
			env:  []string{"X_GOOGLE_TARGET_PLATFORM=flex"},
			want: 0,
		},
		{
			name: "no flex envar set",
			env:  []string{},
			want: 100,
		},
	}

	var files map[string]string
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildpacktest.RunDetectPhaseForTest(t, DetectFn, tc.name, files, tc.env, defaultStack, tc.want)

			if result.ExitCode != tc.want {
				t.Errorf("unexpected exit status %d, want %d", result.ExitCode, tc.want)
				t.Errorf("\ncombined stdout, stderr: %s", result.Output)
			}

			if err == nil && tc.want != 0 {
				t.Errorf("unexpected exit status 0, want %d", tc.want)
				t.Errorf("\ncombined stdout, stderr: %s", result.Output)
			}
		})
	}
}
