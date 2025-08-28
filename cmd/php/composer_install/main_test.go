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

package main

import (
	"fmt"
	"net/http"
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
		env   []string
	}{
		{
			name: "gae with composer.json",
			files: map[string]string{
				"index.php":     "",
				"composer.json": "",
			},
			env: []string{
				"X_GOOGLE_TARGET_PLATFORM=gae",
			},
			want: 0,
		},
		{
			name: "gae without composer.json",
			files: map[string]string{
				"index.php": "",
			},
			env: []string{
				"X_GOOGLE_TARGET_PLATFORM=gae",
			},
			want: 100,
		},
		{
			name: "gcf with composer.json",
			files: map[string]string{
				"index.php":     "",
				"composer.json": "",
			},
			env: []string{
				"X_GOOGLE_TARGET_PLATFORM=gcf",
				"GOOGLE_FUNCTION_TARGET=helloWorld",
			},
			want: 0,
		},
		{
			name: "gcf without composer.json",
			files: map[string]string{
				"index.php": "",
			},
			env: []string{
				"X_GOOGLE_TARGET_PLATFORM=gcf",
				"GOOGLE_FUNCTION_TARGET=helloWorld",
			},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	var (
		expectedHash    = "expected_sha384_hash"
		actualHashCmd   = "php -d 'display_errors = Off' -r"
		runInstallerCmd = fmt.Sprintf("php %s/%s", composerLayer, composerSetup)
	)

	testCases := []struct {
		name                string
		mocks               []*mockprocess.Mock
		wantExitCode        int // 0 if unspecified
		wantCommands        []string
		skippedCommands     []string
		env                 []string
		httpStatusInstaller int
		httpStatusSignature int
	}{
		{
			name: "good composer-setup hash with installation",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithStdout(expectedHash)),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			wantCommands: []string{
				actualHashCmd,
				runInstallerCmd,
				"composer --version " + composerVer,
			},
		},
		{
			name: "custom composer version",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithStdout(expectedHash)),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			env: []string{
				"GOOGLE_COMPOSER_VERSION=2.2.21",
			},
			wantCommands: []string{
				"composer --version 2.2.21",
			},
		},
		{
			name: "bad composer-setup hash with installation",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithStdout("actual_sha384_hash")),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			wantCommands: []string{
				actualHashCmd,
			},
			skippedCommands: []string{
				runInstallerCmd,
			},
			wantExitCode: 1,
		},
		{
			name: "unable to download composer-setup",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithStdout(expectedHash)),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			skippedCommands: []string{
				actualHashCmd,
				runInstallerCmd,
			},
			httpStatusInstaller: http.StatusInternalServerError,
			wantExitCode:        1,
		},
		{
			name: "unable to get expected hash",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithStdout(expectedHash)),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			skippedCommands: []string{
				actualHashCmd,
				runInstallerCmd,
			},
			httpStatusSignature: http.StatusInternalServerError,
			wantExitCode:        1,
		},
		{
			name: "unable to get actual hash",
			mocks: []*mockprocess.Mock{
				mockprocess.New("sha384", mockprocess.WithExitCode(1)),
				mockprocess.New("--filename composer", mockprocess.WithExitCode(0)),
			},
			wantCommands: []string{
				actualHashCmd,
			},
			skippedCommands: []string{
				runInstallerCmd,
			},
			wantExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub the installer
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatusInstaller),
				testserver.WithJSON(`test_file_content`),
				testserver.WithMockURL(&composerSetupURL),
			)

			// stub the signature
			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatusSignature),
				testserver.WithJSON(expectedHash),
				testserver.WithMockURL(&composerSigURL),
			)

			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithExecMocks(tc.mocks...),
				bpt.WithEnvs(tc.env...),
			}

			result, err := bpt.RunBuild(t, BuildFn, opts...)
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
