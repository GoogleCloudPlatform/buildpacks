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

func TestInstallAngularBuildAdaptor(t *testing.T) {
	testCases := []struct {
		name           string
		layerMetadata  map[string]any
		angularVersion string
		mocks          []*mockprocess.Mock
	}{
		{
			name:           "download pinned adapter",
			angularVersion: "17.2.0",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@`+PinnedAngularAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata: map[string]any{},
		},
		{
			name:           "download adaptor not needed since it is cached",
			angularVersion: "17.2.0",
			layerMetadata:  map[string]any{"version": PinnedAngularAdapterVersion},
		},
		{
			name:           "download invalid adaptor falls back to latest",
			angularVersion: "20.0.0",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@20.0`, mockprocess.WithStderr("installed adapter failed")),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@latest`, mockprocess.WithStderr("installed adapter")),
			},
			layerMetadata: map[string]any{},
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
			err := InstallAngularBuildAdaptor(ctx, layer, tc.angularVersion)
			if err != nil {
				t.Fatalf("InstallAngularBuildAdaptor() got error: %v", err)
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
