package main

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
	"github.com/google/go-cmp/cmp"
)

func TestNginxConfig(t *testing.T) {

	testCases := []struct {
		name          string
		runtimeConfig appyaml.RuntimeConfig
		want          nginx.Config
	}{
		{
			name:          "document root configured",
			runtimeConfig: appyaml.RuntimeConfig{DocumentRoot: "web"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace/web",
				AppListenAddress:      "app.sock",
			},
		},
		{
			name:          "document root and front controller configured",
			runtimeConfig: appyaml.RuntimeConfig{DocumentRoot: "web", FrontControllerFile: "app.php"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "app.php",
				Root:                  "/workspace/web",
				AppListenAddress:      "app.sock",
			},
		},
		{
			name:          "default settings",
			runtimeConfig: appyaml.RuntimeConfig{},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace",
				AppListenAddress:      "app.sock",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			got := nginxConfig("", tc.runtimeConfig)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nginxConfig diff (-want, +got):\n%s", diff)
			}
		})
	}
}
