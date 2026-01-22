// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package entrypoint

import (
	"bytes"
	"log"
	"strings"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		want        map[string]string
		wantWarning string
		wantError   bool
	}{
		{
			name:      "empty_file",
			content:   "",
			wantError: true,
		},
		{
			name:    "one_process",
			content: "web: bundle exec rackup",
			want: map[string]string{
				"web": "bundle exec rackup",
			},
		},
		{
			name: "multiple_processes",
			content: `web: bundle exec rackup
worker: bundle exec rake jobs`,
			want: map[string]string{
				"web":    "bundle exec rackup",
				"worker": "bundle exec rake jobs",
			},
		},
		{
			name:    "extra_whitespace",
			content: "web:    bundle exec rackup   ",
			want: map[string]string{
				"web": "bundle exec rackup",
			},
		},
		{
			name:    "duplicate_process_keeps_first",
			content: "web: command1\nweb: command2",
			want: map[string]string{
				"web": "command1",
			},
			wantWarning: "WARNING: Skipping duplicate web process: command2",
		},
		{
			name:    "empty_lines_and_comments",
			content: "# comment\n\nweb: command1",
			want: map[string]string{
				"web": "command1",
			},
		},
		{
			name:      "empty_lines_and_comments_only",
			content:   "# comment\n\n",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			logger := log.New(buf, "", 0)
			ctx := gcp.NewContext(gcp.WithLogger(logger))
			got, err := Parse(ctx, tc.content)
			if tc.wantError {
				if err == nil {
					t.Fatalf("Parse(%q) succeeded, want error", tc.content)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) returned an unexpected error: %v", tc.content, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Parse(%q) returned diff (-want +got):\n%s", tc.content, diff)
			}
			if tc.wantWarning != "" && !strings.Contains(buf.String(), tc.wantWarning) {
				t.Errorf("Parse(%q) log output does not contain warning %q, got %q", tc.content, tc.wantWarning, buf.String())
			}
			if tc.wantWarning == "" && buf.String() != "" {
				t.Errorf("Parse(%q) log output contains unexpected warning, got %q", tc.content, buf.String())
			}
		})
	}
}
