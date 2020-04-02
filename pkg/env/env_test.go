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

package env

import (
	"os"
	"testing"
)

func TestIsDebugMode(t *testing.T) {
	testCases := []struct {
		name    string
		notSet  bool
		value   string
		wantErr bool
		want    bool
	}{
		{
			name:   "not set",
			notSet: true,
		},
		{
			name:    "set to empty",
			wantErr: true,
		},
		{
			name:    "set to bad value",
			value:   "not a bool",
			wantErr: true,
		},
		{
			name:  "set to true",
			value: "true",
			want:  true,
		},
		{
			name:  "set to false",
			value: "false",
			want:  false,
		},
		{
			name:  "set to truthy",
			value: "1",
			want:  true,
		},
		{
			name:  "set to falsey",
			value: "0",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.notSet {
				if err := os.Unsetenv(DebugMode); err != nil {
					t.Fatalf("Failed to unset env: %v", err)
				}
			} else {
				if err := os.Setenv(DebugMode, tc.value); err != nil {
					t.Fatalf("Failed to set env: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(DebugMode); err != nil {
						t.Fatalf("Failed to unset env: %v", err)
					}
				}()
			}

			got, err := IsDebugMode()

			if err != nil != tc.wantErr {
				t.Fatalf("got err=%t, want err=%t: %v", err != nil, tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("IsDebugMode=%t, want=%t", got, tc.want)
			}
		})
	}
}
