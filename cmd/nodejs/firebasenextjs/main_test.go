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

package main

import (
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "with next config",
			files: map[string]string{
				"index.js":       "",
				"next.config.js": "",
			},
			want: 0,
		},
		{
			name: "without next config",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name          string
		wantExitCode  int
		wantCommands  []string
		opts          []bpt.Option
		mocks         []*mockprocess.Mock
		files         map[string]string
		filesExpected map[string]string
	}{
		{
			name: "replace build script",
			files: map[string]string{
				"package.json": `{
				"scripts": {
					"build": "next build"
				},
				"dependencies": {
					"next": "13.0.0"
				}
			}`, "package-lock.json": `{
				"packages": {
					"node_modules/next": {
						"version": "13.0.0"
					}
				}
			}`,
			},
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@`+nodejs.PinnedNextjsAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			wantCommands: []string{
				"npm install --prefix npm_modules @apphosting/adapter-nextjs@" + nodejs.PinnedNextjsAdapterVersion,
			},
		},
		{
			name: "build script doesnt exist",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "13.0.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {
							"version": "13.0.0"
						}
					}
				}`,
			},
		},
		{
			name: "build script already set",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-nextjs-build"
					},
					"dependencies": {
						"next": "13.0.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {
							"version": "13.0.0"
						}
					}
				}`,
			},
		},
		{
			name: "supports versions with constraints",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-nextjs-build"
					},
					"dependencies": {
						"next": "^13.0.0"
					}
				}`, "package-lock.json": `{
					"packages": {
						"node_modules/next": {
							"version": "13.5.6"
						}
					}
				}`,
			},
		},
		{
			name: "error out if the version is below 13.0.0",
			files: map[string]string{
				"package.json": `{
				"dependencies": {
					"next": "12.0.0"
				}
			}`,
				"package-lock.json": `{
				"packages": {
					"node_modules/next": {
						"version": "12.0.0"
					}
				}
			}`,
			},
			wantExitCode: 1,
		},
		{
			name: "read supported concrete version from package-lock.json",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "12.0.0 - 14.0.0"
					}
				}`,
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {
							"version": "14.0.0"
						}
					}
				}`,
			},
		},
		{
			name: "read supported concrete version from pnpm-lock.yaml",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "12.0.0 - 14.0.0"
					}
				}`,
				"pnpm-lock.yaml": `
dependencies:
  next:
    version: 14.0.0(@babel/core@7.23.9)

`,
			},
		},
		{
			name: "read supported concrete version from yaml.lock berry",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "^13.1.0"
					}
				}`,
				"yarn.lock": `
"next@npm:^13.1.0":
	version: 13.5.6`,
			},
		},
		{
			name: "read supported concrete version from yaml.lock classic",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "^13.0.0"
					}
				}`,
				"yarn.lock": `
next@^13.0.0:
  version: "13.5.6"
`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithExecMocks(tc.mocks...),
			}
			opts = append(opts, tc.opts...)
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
			}
		})
	}
}
