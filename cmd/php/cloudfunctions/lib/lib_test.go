// Copyright 2025 Google LLC
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

package lib

import (
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "with composer.json",
			files: map[string]string{
				"index.php":     "",
				"composer.json": "",
			},
			want: 0,
		},
		{
			name: "without composer.json",
			files: map[string]string{
				"index.php": "",
			},
			want: 0,
		},
		{
			name: "without any PHP files",
			files: map[string]string{
				"foo.txt": "",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestEntrypoint(t *testing.T) {
	ep, err := entrypoint(gcp.NewContext())
	if err != nil {
		t.Fatalf("unexpected error creating entrypoint: %v", err)
	}

	want := "serve -enable-dynamic-workers -workers=1024 vendor/google/cloud-functions-framework/router.php"
	if ep.Command != want {
		t.Errorf("entrypoint set wrong, got: %q, want: %q", ep.Command, want)
	}
}
