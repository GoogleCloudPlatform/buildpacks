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
	"testing"
)

func TestExtractVersion(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "gunicorn_correct_version",
			str:  "Version: 19.9.0\n",
			want: "19.9.0",
		},
		{
			name: "gunicorn_misformatted_version",
			str:  "version 18\n",
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match := versionRegexp.FindStringSubmatch(tc.str)
			if len(match) < 2 {
				if tc.want != "" {
					t.Errorf("too few matches, did not capture version")
				}
			} else {
				if match[1] != tc.want {
					t.Errorf("ExtractVersion() got %q, want %q", match[1], tc.want)
				}
			}
		})
	}
}
