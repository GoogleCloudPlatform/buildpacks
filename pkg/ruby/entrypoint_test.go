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

package ruby

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestEntrypoint(t *testing.T) {
	testCases := []struct {
		name    string
		files   []string
		want    string
		wantErr bool
	}{
		{
			name:  "rails no locks",
			files: []string{"bin/rails"},
			want:  "bin/rails server",
		},
		{
			name:  "rails bundle only Gemfile.lock",
			files: []string{"bin/rails", "Gemfile.lock"},
			want:  "bundle exec bin/rails server",
		},
		{
			name:  "rails bundle only gems.locked",
			files: []string{"bin/rails", "gems.locked"},
			want:  "bundle exec bin/rails server",
		},
		{
			name:  "rails bundle both locks",
			files: []string{"bin/rails", "Gemfile.lock", "gems.locked"},
			want:  "bundle exec bin/rails server",
		},
		{
			name:  "rack no locks",
			files: []string{"config.ru"},
			want:  "rackup --port $PORT",
		},
		{
			name:  "rack bundle only Gemfile.lock",
			files: []string{"config.ru", "Gemfile.lock"},
			want:  "bundle exec rackup --port $PORT",
		},
		{
			name:  "rack bundle only gems.locked",
			files: []string{"config.ru", "gems.locked"},
			want:  "bundle exec rackup --port $PORT",
		},
		{
			name:  "rack bundle both locks",
			files: []string{"config.ru", "Gemfile.lock", "gems.locked"},
			want:  "bundle exec rackup --port $PORT",
		},
		{
			name:    "cannot infer",
			files:   []string{"some_file.rb"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "test-entrypoint-")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Fatalf("deleting temp dir %s: %v", tempDir, err)
				}
			}()

			for _, f := range tc.files {
				fn := filepath.Join(tempDir, f)
				if err := os.MkdirAll(path.Dir(fn), 0755); err != nil {
					t.Fatalf("creating dir %s: %v", path.Dir(fn), err)
				}
				if err := ioutil.WriteFile(fn, []byte("content"), 0644); err != nil {
					t.Fatalf("writing file %s: %v", fn, err)
				}
			}
			ctx := gcp.NewContext()

			got, gotErr := InferEntrypoint(ctx, tempDir)

			if gotErr != nil != tc.wantErr {
				t.Fatalf("InferEntrypoint() got err %v, want err %v", gotErr, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("InferEntrypoint() got=%s, want=%s", got, tc.want)
			}
		})
	}
}
