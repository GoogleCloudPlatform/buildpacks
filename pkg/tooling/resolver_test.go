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

package tooling

import (
	"testing"
)

func TestResolveToolVersion(t *testing.T) {
	tests := []struct {
		name           string
		language       string
		toolName       string
		runtimeVersion string
		stackID        string
		want           string
		wantErr        bool
	}{
		{
			name:     "default_python_uv",
			language: "python",
			toolName: "uv",
			want:     "0.11.0",
		},
		{
			name:     "default_java_gradle",
			language: "java",
			toolName: "gradle",
			want:     "9.4.1",
		},
		{
			name:           "java11_specific_gradle",
			language:       "java",
			toolName:       "gradle",
			runtimeVersion: "11.0.1",
			stackID:        "ubuntu1804",
			want:           "8.14.3",
		},
		{
			name:           "python39_specific_poetry",
			language:       "python",
			toolName:       "poetry",
			runtimeVersion: "3.9.25",
			want:           "2.2.1",
		},
		{
			name:     "unknown tool",
			language: "python",
			toolName: "unknown_tool",
			wantErr:  true,
		},
	}
	defer MockData()()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveToolVersion(tc.language, tc.toolName, tc.runtimeVersion, tc.stackID)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveToolVersion() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ResolveToolVersion() = %v, want = %v", got, tc.want)
			}
		})
	}
}
