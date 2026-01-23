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
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "with pubspec",
			files: map[string]string{
				"foo.dart":     "",
				"pubspec.yaml": "",
			},
			want: 0,
		},
		{
			name: "without pubspec",
			files: map[string]string{
				"foo.dart": "",
			},
			want: 0,
		},
		{
			name: "without dart files",
			files: map[string]string{
				"index.txt": "",
			},
			want: 100,
		},
		{
			name: "with flutter",
			files: map[string]string{
				"foo.dart": "",
				"pubspec.yaml": `
name: example_flutter_app

dependencies:
  flutter:
    sdk: flutter
`,
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
