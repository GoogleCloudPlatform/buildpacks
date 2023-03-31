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
	"strings"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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

func TestPhpFpm_DisableDecorateWorkersOutput_ForPhp_Gt_php72(t *testing.T) {
	testCases := []struct {
		name                             string
		runtime                          string
		wantDecorateWorkersOutputEqualNo bool
	}{
		{
			name:                             "runtime is php55, decorate_workers_output unset",
			runtime:                          "php55",
			wantDecorateWorkersOutputEqualNo: false,
		},
		{
			name:                             "runtime is php72, decorate_workers_output unset",
			runtime:                          "php72",
			wantDecorateWorkersOutputEqualNo: false,
		},
		{
			name:                             "runtime is php73, decorate_workers_output set to no",
			runtime:                          "php73",
			wantDecorateWorkersOutputEqualNo: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcpbuildpack.NewContext()
			os.Setenv(env.Runtime, tc.runtime)

			f, err := writeFpmConfig(ctx, os.TempDir())
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
				t.Errorf("Incorrect `strings.Contains(string(cfgBytes), \"decorate_workers_output = no\")` value for runtime %v. got: %v, want: %v", tc.runtime, got, tc.wantDecorateWorkersOutputEqualNo)
			}
			os.Remove(filename)
		})
	}
}
