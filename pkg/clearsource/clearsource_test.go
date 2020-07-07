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
package clearsource

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
)

func TestPathsToRemove(t *testing.T) {
	testCases := []struct {
		name       string
		files      []string
		exclusions []string
		want       []string
	}{
		{
			name:       "Empty files",
			files:      []string{},
			exclusions: []string{"foo"},
			want:       []string{},
		},
		{
			name:       "Empty exclusions",
			files:      []string{"foo"},
			exclusions: []string{},
			want:       []string{"foo"},
		},
		{
			name:       "Exclusion does not overlap",
			files:      []string{"foo"},
			exclusions: []string{"bar"},
			want:       []string{"foo"},
		},
		{
			name:       "Exclusion overlap",
			files:      []string{"foo", "bar"},
			exclusions: []string{"bar"},
			want:       []string{"foo"},
		},
		{
			name:       "Multiple exclusions overlap",
			files:      []string{"foo", "bar", "baz"},
			exclusions: []string{"foo", "baz"},
			want:       []string{"bar"},
		},
		{
			name:       "Files with partial matching",
			files:      []string{"foo.bar", "foo", "bar"},
			exclusions: []string{"foo", "bar"},
			want:       []string{"foo.bar"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			for _, file := range tc.files {
				path := filepath.Join(tDir, file)
				err = ioutil.WriteFile(path, []byte{}, 0644)
				if err != nil {
					t.Fatalf("writing to file %s: %v", path, err)
				}
			}
			ctx := gcp.NewContextForTests(buildpack.Info{}, "")

			got, err := pathsToRemove(ctx, tDir, tc.exclusions)
			if err != nil {
				t.Errorf("pathsToRemove() returned error: %v", err)
			}
			if reflect.DeepEqual(got, tc.want) {
				t.Errorf("pathsToRemove() returned %s want %s", got, tc.want)
			}
		})
	}
}
