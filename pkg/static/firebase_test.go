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

package static

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFirebaseConfig(t *testing.T) {
	ptrBool := func(b bool) *bool { return &b }

	tests := []struct {
		name        string
		fileContent string // Non-empty will create a temp file; empty represents a missing file
		wantConfigs []HostingConfig
		wantErr     bool
	}{
		{
			name:        "not found",
			fileContent: "",
			wantConfigs: nil,
		},
		{
			name: "single hosting object",
			fileContent: `{
				"hosting": {
					"public": "dist",
					"cleanUrls": true,
					"trailingSlash": false,
					"rewrites": [{"source": "**", "destination": "/index.html"}],
					"redirects": [{"source": "/old", "destination": "/new", "type": 301}],
					"headers": [{"source": "**/*.css", "headers": [{"key": "Cache-Control", "value": "max-age=31536000"}]}]
				}
			}`,
			wantConfigs: []HostingConfig{
				{
					Public:        "dist",
					CleanUrls:     true,
					TrailingSlash: ptrBool(false),
					Rewrites: []Rewrite{
						{Source: "**", Destination: "/index.html"},
					},
					Redirects: []Redirect{
						{Source: "/old", Destination: "/new", Type: 301},
					},
					Headers: []Header{
						{
							Source: "**/*.css",
							Headers: []HeaderConfig{
								{Key: "Cache-Control", Value: "max-age=31536000"},
							},
						},
					},
				},
			},
		},
		{
			name: "array hosting objects",
			fileContent: `{
				"hosting": [
					{"target": "app1", "public": "dist1"},
					{"target": "app2", "public": "dist2"}
				]
			}`,
			wantConfigs: []HostingConfig{
				{Target: "app1", Public: "dist1"},
				{Target: "app2", Public: "dist2"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := "nonexistent.json"
			if tc.fileContent != "" {
				tmpDir := t.TempDir()
				p = filepath.Join(tmpDir, "firebase.json")
				if err := os.WriteFile(p, []byte(tc.fileContent), 0644); err != nil {
					t.Fatalf("writing temp file: %v", err)
				}
			}

			got, err := ParseFirebaseConfig(p)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseFirebaseConfig(%q) returned error %v, wantErr %t", p, err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantConfigs, got); diff != "" {
				t.Errorf("ParseFirebaseConfig(%q) returned unexpected diff (-want +got):\n%s", p, diff)
			}
		})
	}
}
