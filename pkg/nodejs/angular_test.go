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
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

func TestInstallAngularBuildAdaptor(t *testing.T) {
	testCases := []struct {
		name          string
		layerMetadata map[string]any
		mocks         []*mockprocess.Mock
	}{
		{
			name: "download adaptor succeeds",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-angular@latest`, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata: map[string]any{},
		},
		{
			name:          "download adaptor not needed since it is cached",
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
			err := InstallAngularBuildAdaptor(ctx, layer)
			if err != nil {
				t.Fatalf("InstallAngularBuildAdaptor() got error: %v", err)
			}
		})
	}
}
