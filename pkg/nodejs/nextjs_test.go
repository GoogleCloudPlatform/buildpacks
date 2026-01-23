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
	"github.com/google/go-cmp/cmp"
	"github.com/buildpacks/libcnb/v2"
)

const (
	mockLatestNextjsAdapterVersion = "1.1.1"
)

func TestInstallNextJsBuildAdaptor(t *testing.T) {
	testCases := []struct {
		name                  string
		layerMetadata         map[string]any
		mocks                 []*mockprocess.Mock
		expectedLayerMetadata map[string]any
	}{
		{
			name:          "adapter download is skipped if version is cached",
			layerMetadata: map[string]any{"version": mockLatestNextjsAdapterVersion},
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-nextjs version", mockprocess.WithStdout(mockLatestNextjsAdapterVersion)),
			},
			expectedLayerMetadata: map[string]any{"version": mockLatestNextjsAdapterVersion},
		},
		{
			name: "download latest adapter version",
			mocks: []*mockprocess.Mock{
				mockprocess.New("npm view @apphosting/adapter-nextjs version", mockprocess.WithStdout(mockLatestNextjsAdapterVersion)),
				mockprocess.New(`npm install --prefix npm_modules @apphosting/adapter-nextjs@`+mockLatestNextjsAdapterVersion, mockprocess.WithStdout("installed adaptor")),
			},
			layerMetadata:         map[string]any{},
			expectedLayerMetadata: map[string]any{"version": mockLatestNextjsAdapterVersion},
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
			if err := InstallNextJsBuildAdapter(ctx, layer); err != nil {
				t.Fatalf("InstallNextJsBuildAdaptor() got error: %v", err)
			}
			if diff := cmp.Diff(tc.expectedLayerMetadata, layer.Metadata); diff != "" {
				t.Errorf("InstallNextJsBuildAdaptor() mismatch in metadata (-want +got):\n%s", diff)
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
