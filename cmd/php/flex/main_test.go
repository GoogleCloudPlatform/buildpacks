package main

import (
	"os"
	"path/filepath"
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
		{
			name:          "document root and nginx http include config given",
			runtimeConfig: appyaml.RuntimeConfig{DocumentRoot: "web", NginxConfInclude: "path/include.conf"},
			want: nginx.Config{
				Port:                  8080,
				FrontControllerScript: "index.php",
				Root:                  "/workspace/web",
				AppListenAddress:      "app.sock",
				NginxConfInclude:      "/workspace/path/include.conf",
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

func TestNginxConfCmdArgs(t *testing.T) {
	tempDir := os.TempDir()
	testCases := []struct {
		name          string
		runtimeConfig appyaml.RuntimeConfig
		want          []string
	}{
		{
			name:          "nginx config overrides the path",
			runtimeConfig: appyaml.RuntimeConfig{NginxConfOverride: "path/override.conf"},
			want:          []string{"--nginxConfigPath", "/workspace/path/override.conf"},
		},
		{
			name:          "default settings",
			runtimeConfig: appyaml.RuntimeConfig{},
			want: []string{
				"--nginxConfigPath", filepath.Join(tempDir, "nginx.conf"),
				"--serverConfigPath", filepath.Join(tempDir, "nginxserver.conf"),
			},
		},
		{
			name:          "nginx http conf included",
			runtimeConfig: appyaml.RuntimeConfig{NginxConfHTTPInclude: "path/include.conf"},
			want: []string{
				"--nginxConfigPath", filepath.Join(tempDir, "nginx.conf"),
				"--serverConfigPath", filepath.Join(tempDir, "nginxserver.conf"),
				"--httpIncludeConfigPath", filepath.Join(defaultRoot, "path/include.conf"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := nginxConfCmdArgs(tempDir, tc.runtimeConfig)
			if err != nil {
				t.Fatalf("nginxConfCmdArgs(%v, %v) failed with err: %v", tempDir, tc.runtimeConfig, err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nginxConfCmdArgs(%v, %v) returned unexpected difference in args (-want, +got):\n%s", tempDir, tc.runtimeConfig, diff)
			}

		})
	}

}
