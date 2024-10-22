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
		envs  []string
		want  int
	}{
		{
			name: "not a firebase apphosting app",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
		{
			name: "with next config",
			files: map[string]string{
				"index.js":       "",
				"next.config.js": "",
				"package.json": `{
					"scripts": {
						"build": "adapter build"
					}
				}`,
				"package-lock.json": `{
					"packages": {
					}
				}`,
			},
			envs: []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 0,
		},
		{
			name: "without next config",
			files: map[string]string{
				"index.js": "",
				"package.json": `{
					"scripts": {
						"build": "adapter build"
					}
				}`,
				"package-lock.json": `{
					"packages": {
					}
				}`,
			},
			envs: []string{"X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 100,
		},
		{
			name: "with next config in app dir",
			files: map[string]string{
				"apps/next-app/index.js":       "",
				"apps/next-app/next.config.js": "",
				"package.json": `{
					"scripts": {
						"build": "adapter build"
					}
				}`,
				"package-lock.json": `{
					"packages": {
					}
				}`,
			},
			envs: []string{"GOOGLE_BUILDABLE=apps/next-app", "X_GOOGLE_TARGET_PLATFORM=fah"},
			want: 0,
		},
		{
			name: "with apphosting:build script",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"apphosting:build": "adapter build"
					}
				}`,
				"package-lock.json": `{
					"packages": {
					}
				}`,
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name                 string
		wantExitCode         int
		files                map[string]string
		shouldInstallAdapter bool
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
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
		},
		{
			name: "adapter already installed",
			files: map[string]string{
				"package.json": `{
					"scripts": {
						"build": "apphosting-adapter-nextjs-build"
					},
					"dependencies": {
						"@apphosting/adapter-nextjs": "14.0.7"
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
			shouldInstallAdapter: false,
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
			shouldInstallAdapter: true,
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
			wantExitCode:         1,
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
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
			shouldInstallAdapter: true,
		},
		{
			name: "read supported concrete version from package.json with unsupported lock file format",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "14.0.0"
					}
				}`,
				"pnpm-lock.yaml": `
unsupported:
  next:
    version: 14.0.0(@babel/core@7.23.9)

`,
			},
			shouldInstallAdapter: true,
		},
		{
			name: "read range version from package.json with unsupported lock file format",
			files: map[string]string{
				"package.json": `{
					"dependencies": {
						"next": "14.0.0-15.0.0"
					}
				}`,
				"pnpm-lock.yaml": `
unsupported:
  next:
    version: 14.0.0(@babel/core@7.23.9)

`,
			},
			shouldInstallAdapter: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithFiles(tc.files),
				bpt.WithExecMocks(
					mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@`+nodejs.PinnedNextjsAdapterVersion, mockprocess.WithStdout("installed adaptor")),
				),
			}
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			wantCommands := []string{
				"npm install --prefix npm_modules @apphosting/adapter-nextjs@" + nodejs.PinnedNextjsAdapterVersion,
			}
			for _, cmd := range wantCommands {
				if result.ExitCode == 0 && !result.CommandExecuted(cmd) && tc.shouldInstallAdapter {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
				if result.ExitCode == 0 && result.CommandExecuted(cmd) && !tc.shouldInstallAdapter {
					t.Errorf("didn't expect command %q to be executed, but it was, build output: %s", cmd, result.Output)
				}
			}
		})
	}
}
