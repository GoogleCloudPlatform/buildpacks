// Copyright 2022 Google LLC
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
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

func TestParseRubyVersion(t *testing.T) {

	type lockFile struct {
		name    string
		content string
	}

	testCases := []struct {
		name       string
		runtimeEnv string
		want       string
		lockFile   lockFile
		wantError  bool
	}{
		{
			name: "from Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
RUBY VERSION
   ruby 2.5.7p206
`},
			want: "2.5.7",
		},
		{
			name: "from Gemfile.lock with jruby",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
RUBY VERSION
		ruby 1.9.3 (jruby 1.6.7)
`,
			},
			wantError: true,
		},
		{
			name: "invalid Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
RUBY VERSION
		809809ruby 2.5.7p206adasdada
`,
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runtimeEnv != "" {
				t.Setenv(env.RuntimeVersion, tc.runtimeEnv)
			}

			tempRoot := t.TempDir()

			path := filepath.Join(tempRoot, tc.lockFile.name)
			if err := ioutil.WriteFile(path, []byte(tc.lockFile.content), 0644); err != nil {
				t.Fatalf("writing file %s: %v", path, err)
			}

			got, err := ParseRubyVersion(path)

			if err != nil && !tc.wantError {
				t.Fatalf("ParseRubyVersion(%q) got error: %v", path, err)
			}

			if err == nil && tc.wantError {
				t.Fatalf("ParseRubyVersion(%q) wanted error, got nil", path)
			}

			if got != tc.want {
				t.Errorf("ParseRubyVersion(file) = %q, want %q", got, tc.want)
			}
		})
	}

}

func TestParseBundlerVersion(t *testing.T) {

	type lockFile struct {
		name    string
		content string
	}

	testCases := []struct {
		name       string
		runtimeEnv string
		want       string
		lockFile   lockFile
		wantError  bool
	}{
		{
			name: "from Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
BUNDLED WITH
   1.17.3
`},
			want: "1.17.3",
		},
		{
			name: "invalid Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
BUNDLED WITH
   invalid something
`,
			},
			wantError: true,
		},
		{
			name: "not specified in Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
`,
			},
			want: "",
		},
		{
			name: "invalid semver in Gemfile.lock",
			lockFile: lockFile{
				name: "Gemfile.lock",
				content: `
BUNDLED WITH
   1.-45.ayj
`,
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runtimeEnv != "" {
				t.Setenv(env.RuntimeVersion, tc.runtimeEnv)
			}

			tempRoot := t.TempDir()

			path := filepath.Join(tempRoot, tc.lockFile.name)
			if err := ioutil.WriteFile(path, []byte(tc.lockFile.content), 0644); err != nil {
				t.Fatalf("writing file %s: %v", path, err)
			}

			got, err := ParseBundlerVersion(path)

			if err != nil && !tc.wantError {
				t.Fatalf("ParseBundlerVersion(%q) got error: %v", path, err)
			}

			if err == nil && tc.wantError {
				t.Fatalf("ParseBundlerVersion(%q) wanted error, got nil", path)
			}

			if got != tc.want {
				t.Errorf("ParseBundlerVersion(%q) = %q, want %q", path, got, tc.want)
			}
		})
	}

}
