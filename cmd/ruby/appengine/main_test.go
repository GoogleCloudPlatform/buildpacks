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
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name:  "GAE target platform",
			files: map[string]string{},
			env:   []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want:  0,
		},
		{
			name:  "non-GAE target platform",
			files: map[string]string{},
			env:   []string{"X_GOOGLE_TARGET_PLATFORM=gcf"},
			want:  100,
		},
		{
			name:  "no target platform",
			files: map[string]string{},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

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
			tempDir, err := os.MkdirTemp("", "test-entrypoint-")
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
				if err := os.WriteFile(fn, []byte("content"), 0644); err != nil {
					t.Fatalf("writing file %s: %v", fn, err)
				}
			}
			ctx := gcp.NewContext()

			got, gotErr := entrypoint(ctx, tempDir)

			if gotErr != nil {
				if gotErr != nil != tc.wantErr {
					t.Errorf("Entrypoint() got err %b, want err %b", gotErr != nil, tc.wantErr)
				}
				return
			}

			want := appstart.Entrypoint{
				Type:    appstart.EntrypointGenerated.String(),
				Command: tc.want,
			}
			if !reflect.DeepEqual(*got, want) {
				t.Errorf("Entrypoint() got=%#v, want=%#v", *got, want)
			}
		})
	}
}
