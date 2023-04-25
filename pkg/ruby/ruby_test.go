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
		name       string
		runtimeEnv string
		want       string
		lockFiles  []lockFile
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
		{
			name: "both Gemfile.lock and .ruby-version are present",
			lockFiles: []lockFile{
				lockFile{
					name:    ".ruby-version",
					content: `3.0.5`,
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
			name: "ruby-version is present",
			lockFiles: []lockFile{
				lockFile{
					name:    ".ruby-version",
					content: `3.2.2`,
				},
			},
			want: "3.2.2",
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

			if err != nil {
				t.Fatalf("DetectRubyVersion(ctx) got error: %v", err)
			}

			if got != tc.want {
				t.Errorf("DetectRubyVersion(ctx) = %q, want %q", got, tc.want)
			}
		})
	}

}

func TestDetectVersionFailures(t *testing.T) {

	type lockFile struct {
		name    string
		content string
	}

	testCases := []struct {
		name         string
		runtimeEnv   string
		lockFiles    []lockFile
		errorContent string
	}{
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
			errorContent: "Ruby version \"2.5.7\" in Gemfile.lock can't be overriden to \"3.0.1\" using GOOGLE_RUNTIME_VERSION environment variable",
		},
		{
			name:       "from .ruby-version with different version on env",
			runtimeEnv: "3.0.1",
			lockFiles: []lockFile{
				lockFile{
					name:    ".ruby-version",
					content: `3.0.5`,
				},
			},
			errorContent: "There is a conflict between Ruby versions specified in .ruby-version file and the GOOGLE_RUNTIME_VERSION environment variable. Please resolve the conflict by choosing only one way to specify the ruby version",
		},
		{
			name: "from Gemfile.lock with different version in .ruby-version",
			lockFiles: []lockFile{
				lockFile{
					name: "Gemfile.lock",
					content: `
RUBY VERSION
   ruby 2.5.7p206
`},
				lockFile{
					name:    ".ruby-version",
					content: `3.0.5`,
				},
			},
			errorContent: "There is a conflict between the Ruby version \"2.5.7\" in Gemfile.lock and \"3.0.5\" in .ruby-version file.Please resolve the conflict by choosing only one way to specify the ruby version.",
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
			_, err := DetectVersion(ctx)

			if err == nil {
				t.Fatal("DetectRubyVersion(ctx) wanted error, got nil")
			}

			if tc.errorContent != "" && !strings.Contains(err.Error(), tc.errorContent) {
				t.Fatalf("DetectRubyVersion(ctx) got error: %v, wanted %v", err.Error(), tc.errorContent)
			}
		})
	}
}

func TestSupportsBundler1(t *testing.T) {
	testCases := []struct {
		name        string
		rubyVersion string
		want        bool
	}{
		{
			name:        "2.x",
			rubyVersion: "2.7.6",
			want:        true,
		},
		{
			name:        "3.0.x",
			rubyVersion: "3.0.5",
			want:        true,
		},
		{
			name:        "3.1.x",
			rubyVersion: "3.1.2",
			want:        true,
		},
		{
			name:        "3.2.x",
			rubyVersion: "3.2.0",
			want:        false,
		},
		{
			name:        "future versions",
			rubyVersion: "4.3.0",
			want:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))

			if tc.rubyVersion != "" {
				t.Setenv(RubyVersionKey, tc.rubyVersion)
			}
			got, err := SupportsBundler1(ctx)

			if err != nil {
				t.Fatalf("SupportsBundler1(ctx) got error: %v", err)
			}

			if got != tc.want {
				t.Errorf("SupportsBundler1(ctx) = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestSupportsBundler1Failures(t *testing.T) {
	testCases := []struct {
		name         string
		rubyVersion  string
		errorContent string
	}{
		{
			name:        "invalid ruby version",
			rubyVersion: "invalid version",
		},
		{
			name:         "ruby version not set",
			rubyVersion:  "",
			errorContent: "Invalid Semantic Version",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))

			if tc.rubyVersion != "" {
				t.Setenv(RubyVersionKey, tc.rubyVersion)
			}
			_, err := SupportsBundler1(ctx)

			if err == nil {
				t.Fatal("SupportsBundler1(ctx) wanted error, got nil")
			}

			if tc.errorContent != "" && !strings.Contains(err.Error(), tc.errorContent) {
				t.Fatalf("SupportsBundler1(ctx) got error: %v, wanted %v", err.Error(), tc.errorContent)
			}
		})
	}
}
