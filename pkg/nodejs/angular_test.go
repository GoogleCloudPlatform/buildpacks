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
	"encoding/json"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
	"github.com/buildpacks/libcnb/v2"
)

const (
	mockLatestAngularAdapterVersion = "17.2.14"
)

func TestInstallAngularBuildAdaptor(t *testing.T) {
	testCases := []struct {
		name                  string
		layerMetadata         map[string]any
		angularVersion        string
		mocks                 []*mockprocess.Mock
		expectedLayerMetadata map[string]any
	}{
		{
			name:          "download adaptor not needed since it is cached",
			layerMetadata: map[string]any{"version": mockLatestAngularAdapterVersion},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
			},
			expectedLayerMetadata: map[string]any{"version": mockLatestAngularAdapterVersion},
		},
		{
			name: "download pinned adapter",
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-angular version", mockprocess.WithStdout(mockLatestAngularAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+mockLatestAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata:         map[string]any{},
			expectedLayerMetadata: map[string]any{"version": mockLatestAngularAdapterVersion},
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
			if err := InstallAngularBuildAdapter(ctx, layer); err != nil {
				t.Fatalf("InstallAngularBuildAdaptor() got unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expectedLayerMetadata, layer.Metadata); diff != "" {
				t.Errorf("InstallAngularBuildAdaptor() mismatch in metadata (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractAngularStartCommand(t *testing.T) {
	testsCases := []struct {
		name string
		pjs  string
		want string
	}{
		{
			name: "with angular serve command",
			pjs: `{
					"scripts": {
						"ng": "ng",
						"start": "ng serve",
						"build": "ng build",
						"watch": "ng build --watch --configuration development",
						"test": "ng test",
						"serve:ssr:my-angular-app": "node dist/my-angular-app/server/server.mjs"
					}
				}`,
			want: "node dist/my-angular-app/server/server.mjs",
		},
		{
			name: "no angular serve command",
			pjs: `{
					"main": "main.js",
					"scripts": {
						"start": "node main.js"
					}
				}`,
			want: "",
		},
		{
			name: "no scripts",
			pjs:  `{}`,
			want: "",
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			var pjs *PackageJSON = nil
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}
			got := ExtractAngularStartCommand(pjs)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractAngularStartCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
