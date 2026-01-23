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
	"fmt"
	"net/http"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "both gems.rb and Gemfile",
			files: map[string]string{
				"gems.rb": "",
				"Gemfile": "",
			},
			want: 0,
		},
		{
			name: "only gems.rb",
			files: map[string]string{
				"gems.rb": "",
			},
			want: 0,
		},
		{
			name: "only Gemfile",
			files: map[string]string{
				"Gemfile": "",
			},
			want: 0,
		},
		{
			name:  "neither gems.rb nor Gemfile",
			files: map[string]string{},
			want:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	var (
		installCommand         = fmt.Sprintf("ruby setup.rb -E --no-document --destdir %s --prefix /", layerName)
		bundler1InstallCommand = fmt.Sprintf("gem install bundler:%s --no-document", bundler1Version)
	)

	testCases := []struct {
		name                string
		tarFile             string
		mocks               []*mockprocess.Mock
		wantExitCode        int // 0 if unspecified
		wantCommands        []string
		skippedCommands     []string
		httpStatusInstaller int
		app                 string
		rubyVersion         string
	}{
		{
			name: "bundler 1 in Gemfile.lock",
			mocks: []*mockprocess.Mock{
				mockprocess.New("^ruby"),
				mockprocess.New("^gem"),
			},
			wantCommands: []string{
				installCommand,
				bundler1InstallCommand,
			},
			tarFile: "testdata/dummy-rubygems.tar.gz",
			app:     "testdata/bundler1",
		},
		{
			name: "bundler 2 in Gemfile.lock",
			mocks: []*mockprocess.Mock{
				mockprocess.New("^ruby"),
				mockprocess.New("^cp"),
			},
			wantCommands: []string{
				installCommand,
			},
			tarFile: "testdata/dummy-rubygems.tar.gz",
			app:     "testdata/bundler2",
		},
		{
			// Bundler 1 does not support Ruby 3.2
			name: "Ruby32 does not install bundler 1",
			mocks: []*mockprocess.Mock{
				mockprocess.New("^ruby"),
				mockprocess.New("^cp"),
			},
			wantCommands: []string{
				installCommand,
			},
			skippedCommands: []string{
				bundler1InstallCommand,
			},
			tarFile:     "testdata/dummy-rubygems.tar.gz",
			app:         "testdata/bundler1",
			rubyVersion: "3.2.0",
		},
		{
			name:                "handles download failure",
			httpStatusInstaller: http.StatusNotFound,
			wantExitCode:        1,
		},
		{
			name: "handles rubygems install failure",
			mocks: []*mockprocess.Mock{
				mockprocess.New("^ruby", mockprocess.WithExitCode(1)),
			},
			wantCommands: []string{
				installCommand,
			},
			tarFile:      "testdata/dummy-rubygems.tar.gz",
			app:          "testdata/bundler2",
			wantExitCode: 1,
		},
		{
			name: "handles bundler 1 install failure",
			mocks: []*mockprocess.Mock{
				mockprocess.New("^ruby", mockprocess.WithExitCode(0)),
				mockprocess.New("^gem", mockprocess.WithExitCode(1)),
			},
			wantCommands: []string{
				installCommand,
				bundler1InstallCommand,
			},
			tarFile:      "testdata/dummy-rubygems.tar.gz",
			app:          "testdata/bundler1",
			wantExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatusInstaller),
				testserver.WithFile(testdata.MustGetPath(tc.tarFile)),
				testserver.WithMockURL(&rubygemsURL),
			)

			opts := []buildpacktest.Option{
				buildpacktest.WithTestName(tc.name),
				buildpacktest.WithExecMocks(tc.mocks...),
			}

			if tc.app != "" {
				opts = append(opts, buildpacktest.WithApp(testdata.MustGetPath(tc.app)))
			}

			// Set default Ruby version
			if tc.rubyVersion == "" {
				tc.rubyVersion = "3.0.5"
			}

			t.Setenv(ruby.RubyVersionKey, tc.rubyVersion)
			result, err := buildpacktest.RunBuild(t, BuildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, result: %#v", err, result)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not", cmd)
				}
			}

			for _, cmd := range tc.skippedCommands {
				if result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to not be executed, but it was", cmd)
				}
			}
		})
	}
}
