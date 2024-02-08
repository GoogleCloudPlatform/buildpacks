// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package nodejs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

func TestInstallNextJsBuildAdaptor(t *testing.T) {
	testCases := []struct {
		name          string
		nextjsVersion string
		layerMetadata map[string]any
		mocks         []*mockprocess.Mock
	}{
		{
			name:          "download latest adaptor succeeds",
			nextjsVersion: "latest",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@latest`, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata: map[string]any{},
		},
		{
			name:          "download v13.0 adaptor succeeds",
			nextjsVersion: "13.0.0",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@13.0`, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata: map[string]any{},
		},
		{
			name:          "download invalid adaptor falls back to latest",
			nextjsVersion: "a.b.c",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@latest`, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata: map[string]any{},
		},
		{
			name:          "download adaptor not needed since it is cached",
			nextjsVersion: "latest",
			layerMetadata: map[string]any{"version": "latest"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext(getContextOpts(t, tc.mocks)...)
			layer := &libcnb.Layer{
				Name:     "njsl",
				Path:     t.TempDir(),
				Metadata: tc.layerMetadata,
			}
			err := InstallNextJsBuildAdaptor(ctx, layer, tc.nextjsVersion)
			if err != nil {
				t.Fatalf("InstallNextJsBuildAdaptor() got error: %v", err)
			}
		})
	}
}
func TestDetectNextjsAdaptorVersion(t *testing.T) {
	testCases := []struct {
		name           string
		version        string
		expectedOutput string
		mocks          []*mockprocess.Mock
	}{
		{
			name:           "concrete version",
			version:        "13.0.0",
			expectedOutput: "13.0",
		},
		{
			name:           "handles ^",
			version:        "^13.1.14",
			expectedOutput: "^13.1",
		},
		{
			name:           "handles ~",
			version:        "~13.1.14",
			expectedOutput: "~13.1",
		},
		{
			name:           "handles x patch",
			version:        "13.0.x",
			expectedOutput: "13.0",
		},
		{
			name:           "handles x minor",
			version:        "13.x",
			expectedOutput: "13.x",
		},
		{
			name:           "handles x major",
			version:        "x",
			expectedOutput: "x",
		},
		{
			name:           "handles no patch",
			version:        "13.0",
			expectedOutput: "13.0",
		},
		{
			name:           "handles no minor",
			version:        "13",
			expectedOutput: "13",
		},
		{
			name:           "handles prereleases",
			version:        "13.0.1-canary",
			expectedOutput: "13.0",
		},
		{
			name:           "defaults to latest when version is invalid",
			version:        "a.b.c",
			expectedOutput: "latest",
		},
		{
			name:           "defaults to latest when version is invalid range",
			version:        "13.0.0 - a",
			expectedOutput: "latest",
		},
		{
			name:           "- range translates to proper range",
			version:        "13.0.0 - 14.0.0",
			expectedOutput: ">=13.0 <=14.0",
		},
		{
			name:           "- with x",
			version:        "13.0.0 - 14.x.x",
			expectedOutput: ">=13.0 <=14.x",
		},
		{
			name:           "- with x minor",
			version:        "13.0.0 - 14.x",
			expectedOutput: ">=13.0 <=14.x",
		},
		{
			name:           "- with major",
			version:        "13.0.0 - 14",
			expectedOutput: ">=13.0 <=14",
		},
		{
			name:           "range constraint translates",
			version:        ">13.1.14 <14.1.15",
			expectedOutput: ">=13.1 <=14.1",
		},
		{
			name:           "< translates to <=",
			version:        "<14.0.15",
			expectedOutput: "<=14.0",
		},
		{
			name:           "> translates to >=",
			version:        ">14.0.15",
			expectedOutput: ">=14.0",
		},
		{
			name:           "< maps range with no patch",
			version:        "<14.1",
			expectedOutput: "<14.1",
		},
		{
			name:           "< maps with no minor",
			version:        "<14",
			expectedOutput: "<14",
		},
		{
			name:           "< map properly with 0 patch version",
			version:        ">13.1.14 <14.1.0",
			expectedOutput: ">=13.1 <14.1",
		},
		{
			name:           "<= maps",
			version:        ">13.1.14 <=14.1.15",
			expectedOutput: ">=13.1 <=14.1",
		},
		{
			name:           "handles ^ and <",
			version:        "^13.1.14 <14.1.15",
			expectedOutput: "^13.1 <=14.1",
		},
		{
			name:           "handles ||",
			version:        "^13.1.14 <13.1.15 || >14",
			expectedOutput: "^13.1 <=13.1 || >14",
		},
		{
			name:           "handles prerelease ranges",
			version:        ">13.0.1-canary <14.1.0-canary",
			expectedOutput: ">=13.0 <14.1",
		},
		{
			name:           "handles mixed range",
			version:        "^13.1.14 ~12.0.0-canary <13.1.15-test || >14",
			expectedOutput: "^13.1 ~12.0 <=13.1 || >14",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := detectNextjsAdaptorVersion(tc.version)
			if output != tc.expectedOutput {
				t.Fatalf("detectNextjsAdaptorVersion(%s) output: %s doesn't match expected output %s", tc.version, output, tc.expectedOutput)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	testCases := []struct {
		name            string
		pjs             PackageJSON
		files           map[string]string
		expectedVersion string
		mocks           []*mockprocess.Mock
	}{
		{
			name: "Parses package-lock version",
			pjs: PackageJSON{
				Dependencies: map[string]string{
					"next": "^13.1.0",
				},
			},
			files: map[string]string{
				"package-lock.json": `{
					"packages": {
						"node_modules/next": {
							"version": "13.5.6"
						}
					}
				}`,
			},
			expectedVersion: "13.5.6",
		},
		{
			name: "Parses pnpm-lock version",
			pjs: PackageJSON{
				Dependencies: map[string]string{
					"next": "^13.1.0",
				},
			},
			files: map[string]string{
				"pnpm-lock.yaml": `
dependencies:
  next:
    version: 13.5.6(@babel/core@7.23.9)

`,
			},
			expectedVersion: "13.5.6",
		},
		{
			name: "Parses yarn.lock berry version",
			pjs: PackageJSON{
				Dependencies: map[string]string{
					"next": "^13.1.0",
				},
			},
			files: map[string]string{
				"yarn.lock": `
"next@npm:^13.1.0":
	version: 13.5.6`,
			},
			expectedVersion: "13.5.6",
		},
		{
			name: "Parses yarn.lock classic version",
			pjs: PackageJSON{
				Dependencies: map[string]string{
					"next": "^13.1.0",
				},
			},
			files: map[string]string{
				"yarn.lock": `
next@^13.1.0:
	version: "13.5.6"
				`,
			},
			expectedVersion: "13.5.6",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := os.TempDir()

			ctx := gcp.NewContext(append(getContextOpts(t, tc.mocks), gcp.WithApplicationRoot(tmpDir))...)

			for file, content := range tc.files {
				fn := filepath.Join(ctx.ApplicationRoot(), file)
				ioutil.WriteFile(fn, []byte(content), 0644)
			}

			version := Version(ctx, &tc.pjs)
			if version != tc.expectedVersion {
				t.Fatalf("Version output: %s doesn't match expected output %s", version, tc.expectedVersion)
			}
		})
	}
}

func getContextOpts(t *testing.T, mocks []*mockprocess.Mock) []gcp.ContextOption {
	t.Helper()
	opts := []gcp.ContextOption{}

	// Mock out calls to ctx.Exec, if specified
	if len(mocks) > 0 {
		eCmd, err := mockprocess.NewExecCmd(mocks...)
		if err != nil {
			t.Fatalf("error creating mock exec command: %v", err)
		}
		opts = append(opts, gcp.WithExecCmd(eCmd))
	}
	return opts
}
