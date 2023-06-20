// Copyright 2022 Google LLC
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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/webconfig"
	"github.com/google/go-cmp/cmp"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "php files",
			files: map[string]string{
				"index.php": "",
			},
			want: 0,
		},
		{
			name: "composer.json and php file",
			files: map[string]string{
				"composer.json": "",
				"index.php":     "",
			},
			want: 0,
		},
		{
			name:  "no composer.json and no php files",
			files: map[string]string{},
			want:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, nil, tc.want)
		})
	}
}

func TestPhpFpm_DisableDecorateWorkersOutput_ForPhp_Gte_730(t *testing.T) {
	testCases := []struct {
		name                             string
		version                          string
		wantDecorateWorkersOutputEqualNo bool
	}{
		{
			name:                             "runtime is 5.5.3, decorate_workers_output unset",
			version:                          "5.5.3",
			wantDecorateWorkersOutputEqualNo: false,
		},
		{
			name:                             "runtime is 7.2.9, decorate_workers_output unset",
			version:                          "7.2.9",
			wantDecorateWorkersOutputEqualNo: false,
		},
		{
			name:                             "runtime is 7.3.0, decorate_workers_output set to no",
			version:                          "7.3.0",
			wantDecorateWorkersOutputEqualNo: true,
		},
		{
			name:                             "runtime is 7.3.1, decorate_workers_output set to no",
			version:                          "7.3.1",
			wantDecorateWorkersOutputEqualNo: true,
		},
		{
			name:                             "runtime is 8.1.0, decorate_workers_output set to no",
			version:                          "8.1.0",
			wantDecorateWorkersOutputEqualNo: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithStackID("google"))

			os.Setenv(env.RuntimeVersion, tc.version)

			f, err := writeFpmConfig(ctx, os.TempDir(), webconfig.OverrideProperties{})
			if err != nil {
				t.Fatalf("Encountered an error generating FPM config: %v", err)
			}
			filename := f.Name()
			f.Close()

			cfgBytes, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatalf("Could not read conf file, %v: %v", filename, err)
			}
			got := strings.Contains(string(cfgBytes), "decorate_workers_output = no")
			if got != tc.wantDecorateWorkersOutputEqualNo {
				t.Errorf("Incorrect `strings.Contains(string(cfgBytes), \"decorate_workers_output = no\")` value for runtime PHP %v. got: %v, want: %v", tc.version, got, tc.wantDecorateWorkersOutputEqualNo)
			}
			os.Remove(filename)
		})
	}
}

func TestAddNginxConfCmdArgs(t *testing.T) {
	tempDir := t.TempDir()
	testCases := []struct {
		isFlex    bool
		name      string
		overrides webconfig.OverrideProperties
		want      []string
	}{
		{
			name:      "nginx config overrides the path",
			overrides: webconfig.OverrideProperties{NginxConfOverride: true, NginxConfOverrideFileName: "override.conf"},
			want: []string{
				"--customAppSocket", filepath.Join(tempDir, "app.sock"),
				"--nginxConfigPath", "override.conf"},
		},
		{
			name:      "default settings",
			overrides: webconfig.OverrideProperties{},
			want: []string{
				"--customAppSocket", filepath.Join(tempDir, "app.sock"),
				"--nginxConfigPath", filepath.Join(tempDir, "nginx.conf"),
				"--serverConfigPath", filepath.Join(tempDir, "nginxserver.conf"),
			},
		},
		{
			name:      "default settings for flex applications",
			isFlex:    true,
			overrides: webconfig.OverrideProperties{},
			want: []string{
				"--customAppPort", "9000",
				"--nginxConfigPath", filepath.Join(tempDir, "nginx.conf"),
				"--serverConfigPath", filepath.Join(tempDir, "nginxserver.conf"),
			},
		},
		{
			name:      "nginx http conf included",
			overrides: webconfig.OverrideProperties{NginxHTTPInclude: true, NginxHTTPIncludeFileName: "include.conf"},
			want: []string{
				"--customAppSocket", filepath.Join(tempDir, "app.sock"),
				"--nginxConfigPath", filepath.Join(tempDir, "nginx.conf"),
				"--serverConfigPath", filepath.Join(tempDir, "nginxserver.conf"),
				"--httpIncludeConfigPath", "include.conf",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isFlex {
				os.Setenv("X_GOOGLE_TARGET_PLATFORM", "flex")
			}
			got, err := addNginxConfCmdArgs(tempDir, filepath.Join(tempDir, "nginxserver.conf"), tc.overrides)
			if err != nil {
				t.Fatalf("nginxConfCmdArgs(%v, %v) failed with err: %v", tempDir, tc.overrides, err)
			}

			os.Setenv("X_GOOGLE_TARGET_PLATFORM", "")

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nginxConfCmdArgs(%v, %v) returned unexpected difference in args (-want, +got):\n%s", tempDir, tc.overrides, diff)
			}

		})
	}

}
