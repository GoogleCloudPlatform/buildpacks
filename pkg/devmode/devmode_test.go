// Copyright 2020 Google LLC
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

package devmode

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

func TestWriteAndRunScripts(t *testing.T) {
	testDirRoot, err := ioutil.TempDir("", "test-layer-")
	if err != nil {
		t.Fatalf("Creating temp directory: %v", err)
	}

	testCases := []struct {
		name            string
		config          Config
		layerRoot       string
		wantBuildAndRun string
		wantWatchAndRun string
	}{
		{
			name: "noBuildCmd",
			config: Config{
				BuildCmd: nil,
				RunCmd:   []string{"run-me.sh"},
				Ext:      []string{".js"},
			},
			layerRoot:       filepath.Join(testDirRoot, "noBuildCmd"),
			wantBuildAndRun: "#!/bin/sh\nrun-me.sh",
			wantWatchAndRun: fmt.Sprintf("#!/bin/sh\nwatchexec -r -e .js %s", filepath.Join(testDirRoot, "noBuildCmd", "bin", "build_and_run.sh")),
		},
		{
			name: "withBuildAndRun",
			config: Config{
				BuildCmd: []string{"build-me.sh"},
				RunCmd:   []string{"run-me.sh"},
				Ext:      []string{".cc"},
			},
			layerRoot:       filepath.Join(testDirRoot, "withBuildAndRun"),
			wantBuildAndRun: "#!/bin/sh\nbuild-me.sh && run-me.sh",
			wantWatchAndRun: fmt.Sprintf("#!/bin/sh\nwatchexec -r -e .cc %s", filepath.Join(testDirRoot, "withBuildAndRun", "bin", "build_and_run.sh")),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = os.Mkdir(tc.layerRoot, os.FileMode(0755))
			if err != nil {
				t.Fatalf("Creating temp directory: %v", err)
			}
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tc.layerRoot))
			l := &libcnb.Layer{Path: tc.layerRoot}

			writeBuildAndRunScript(ctx, l, tc.config)

			bar := filepath.Join(tc.layerRoot, "bin", "build_and_run.sh")
			war := filepath.Join(tc.layerRoot, "bin", "watch_and_run.sh")
			c, err := ioutil.ReadFile(bar)
			if err != nil {
				t.Fatal(err)
			}
			if string(c) != tc.wantBuildAndRun {
				t.Errorf("build_and_run.sh = %q, want %q", string(c), tc.wantBuildAndRun)
			}

			c, err = ioutil.ReadFile(war)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantWatchAndRun != string(c) {
				t.Errorf("watch_and_run.sh = %q, want %q", string(c), tc.wantWatchAndRun)
			}
		})
	}
}
