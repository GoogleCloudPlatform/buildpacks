// Copyright 2025 Google LLC
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

package lib

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/webconfig"
	"github.com/google/go-cmp/cmp"
)

func TestNginxConfig(t *testing.T) {

	testCases := []struct {
		name             string
		nginxConfInclude bool
		overrides        webconfig.OverrideProperties
		want             nginx.Config
	}{
		{
			name:      "document root configured",
			overrides: webconfig.OverrideProperties{DocumentRoot: "web"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace/web",
				AppListenAddress:      defaultAddress,
			},
		},
		{
			name:      "document root and front controller configured",
			overrides: webconfig.OverrideProperties{DocumentRoot: "web", FrontController: "app.php"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "app.php",
				Root:                  "/workspace/web",
				AppListenAddress:      defaultAddress,
			},
		},
		{
			name:      "default settings",
			overrides: webconfig.OverrideProperties{},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace",
				AppListenAddress:      defaultAddress,
			},
		},
		{
			name:             "document root and nginx http include config given",
			nginxConfInclude: true,
			overrides:        webconfig.OverrideProperties{DocumentRoot: "web", NginxServerConfInclude: true, NginxServerConfIncludeFileName: "/workspace/include.conf"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace/web",
				AppListenAddress:      defaultAddress,
				NginxConfInclude:      "/workspace/include.conf",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			got := nginxConfig("", tc.overrides)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nginxConfig diff (-want, +got):\n%s", diff)
			}

		})
	}
}
