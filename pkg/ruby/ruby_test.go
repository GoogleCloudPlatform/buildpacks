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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetectVersion(t *testing.T) {

	type lockFile struct {
		name    string
		content string
	}

	testCases := []struct {
		name         string
		runtimeEnv   string
		want         string
		lockFiles    []lockFile
		wantError    bool
		errorContent string
	}{
		{
			name:       "from environment",
			runtimeEnv: "1.1.1",
			want:       "1.1.1",
		},
		{
			name: "from Gemfile.lock",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
   ruby 2.5.7p206
`},
			},
			want: "2.5.7",
		},
		{
			name:       "from Gemfile.lock with same version on env",
			runtimeEnv: "3.0.1",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
   ruby 3.0.1p0
`},
			},
			want: "3.0.1",
		},
		{
			name:       "from Gemfile.lock with different version on env",
			runtimeEnv: "3.0.1",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
   ruby 2.5.7p206
`},
			},
			wantError:    true,
			errorContent: "Ruby version \"2.5.7\" in Gemfile.lock can't be overriden to \"3.0.1\" using GOOGLE_RUNTIME_VERSION environment variable",
		},
		{
			name: "from Gemfile.lock without ruby version",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
PLATFORMS
	ruby

DEPENDENCIES
	sinatra (~> 2.0)

BUNDLED WITH
		1.17.1
`,
				},
			},
			want: defaultVersion,
		},
		{
			name: "from Gemfile.lock with jruby",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
		ruby 1.9.3 (jruby 1.6.7)
`,
				},
			},
			wantError: true,
		},
		{
			name: "invalid Gemfile.lock",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
		809809ruby 2.5.7p206adasdada
`,
				},
			},
			wantError: true,
		},
		{
			name: "from gems.locked",
			lockFiles: []lockFile{
				lockFile{
					name: "gems.locked",
					content: `
RUBY VERSION
   ruby 2.5.7p206
`},
			},
			want: "2.5.7",
		},
		{
			name: "from gems.locked without ruby version",
			lockFiles: []lockFile{
				lockFile{
					name: "gems.locked",
					content: `
PLATFORMS
	ruby

DEPENDENCIES
	sinatra (~> 2.0)

BUNDLED WITH
		1.17.1
`,
				},
			},
			want: defaultVersion,
		},
		{
			name: "from gems.locked with jruby",
			lockFiles: []lockFile{
				lockFile{
					name: "gems.locked",
					content: `
RUBY VERSION
		ruby 1.9.3 (jruby 1.6.7)
`,
				},
			},
			wantError: true,
		},
		{
			name: "invalid gems.locked",
			lockFiles: []lockFile{
				lockFile{
					name: "gems.locked",
					content: `
RUBY VERSION
		809809ruby 2.5.7p206adasdada
`,
				},
			},
			wantError: true,
		},
		{
			name: "both Gemfile.lock and gems.locked is present",
			lockFiles: []lockFile{
				lockFile{
					name: "gems.locked",
					content: `
RUBY VERSION
		ruby 2.5.7p206adasdada
`,
				},
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
		ruby 3.0.5p34
`,
				},
			},
			want: "3.0.5",
		},
		{
			name: "default version",
			want: defaultVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runtimeEnv != "" {
				t.Setenv(env.RuntimeVersion, tc.runtimeEnv)
			}

			tempRoot := t.TempDir()

			if len(tc.lockFiles) > 0 {
				for _, lockFile := range tc.lockFiles {
					path := filepath.Join(tempRoot, lockFile.name)
					if err := ioutil.WriteFile(path, []byte(lockFile.content), 0644); err != nil {
						t.Fatalf("writing file %s: %v", path, err)
					}
				}
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))
			got, err := DetectVersion(ctx)

			if err != nil && !tc.wantError {
				t.Fatalf("DetectRubyVersion(ctx) got error: %v", err)
			}

			if err == nil && tc.wantError {
				t.Fatal("DetectRubyVersion(ctx) wanted error, got nil")
			}

			if tc.errorContent != "" && !strings.Contains(err.Error(), tc.errorContent) {
				t.Fatalf("DetectRubyVersion(ctx) got error: %v, wanted %v", err.Error(), tc.errorContent)
			}

			if got != tc.want {
				t.Errorf("DetectRubyVersion(ctx) = %q, want %q", got, tc.want)
			}
		})
	}

}
