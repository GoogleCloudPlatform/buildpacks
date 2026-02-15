// Copyright 2026 Google LLC
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

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		envs  []string
		files map[string]string
		want  int
	}{
		{
			name: "with_lock_alpha",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lock":     "",
			},
			envs: []string{
				env.ReleaseTrack + "=ALPHA",
			},
			want: 0,
		},
		{
			name: "with_lock_beta",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lock":     "",
			},
			envs: []string{
				env.ReleaseTrack + "=BETA",
			},
			want: 0,
		},
		{
			name: "with_lock",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lock":     "",
			},
			envs: []string{},
			want: 100,
		},
		{
			name: "with_lockb_alpha",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lockb":    "",
			},
			envs: []string{
				env.ReleaseTrack + "=ALPHA",
			},
			want: 0,
		},
		{
			name: "with_lockb_beta",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lockb":    "",
			},
			envs: []string{
				env.ReleaseTrack + "=BETA",
			},
			want: 0,
		},
		{
			name: "with_lockb",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
				"bun.lockb":    "",
			},
			envs: []string{},
			want: 100,
		},
		{
			name: "package_no_lock_alpha",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			envs: []string{
				env.PackageManager + "=bun",
				env.ReleaseTrack + "=ALPHA",
			},
			want: 0,
		},
		{
			name: "package_no_lock_beta",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			envs: []string{
				env.PackageManager + "=bun",
				env.ReleaseTrack + "=BETA",
			},
			want: 0,
		},
		{
			name: "package_no_lock",
			files: map[string]string{
				"index.js":     "",
				"package.json": "",
			},
			envs: []string{
				env.PackageManager + "=bun",
			},
			want: 100,
		},
		{
			name: "without_package",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, DetectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	originalInstallBun := installBun
	defer func() { installBun = originalInstallBun }()
	installBun = func(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
		return nil
	}
	testCases := []struct {
		name              string
		app               string
		envs              []string
		opts              []bpt.Option
		mocks             []*mockprocess.Mock
		wantExitCode      int
		wantCommands      []string
		doNotWantCommands []string
		files             map[string]string
	}{
		{
			name: "bun.lockb_exists_frozen_lockfile",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bun install --frozen-lockfile`, mockprocess.WithExitCode(0)),
			},
			files: map[string]string{
				"package.json": "{}",
				"bun.lockb":    "binary-content",
			},
			wantCommands: []string{
				"bun install --frozen-lockfile",
			},
		},
		{
			name: "bun.lock_exists_frozen_lockfile",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bun install --frozen-lockfile`, mockprocess.WithExitCode(0)),
			},
			files: map[string]string{
				"package.json": "{}",
				"bun.lock":     "lock-content",
			},
			wantCommands: []string{
				"bun install --frozen-lockfile",
			},
		},
		{
			name: "no_lockfile",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bun install`, mockprocess.WithExitCode(0)),
			},
			files: map[string]string{
				"package.json": "{}",
			},
			wantCommands: []string{
				"bun install",
			},
			doNotWantCommands: []string{
				"bun install --frozen-lockfile",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithApp(tc.app),
				bpt.WithEnvs(tc.envs...),
				bpt.WithExecMocks(tc.mocks...),
				bpt.WithFiles(tc.files),
			}
			opts = append(opts, tc.opts...)
			result, err := bpt.RunBuild(t, BuildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}

			if result.ExitCode != tc.wantExitCode {
				t.Errorf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}

			for _, cmd := range tc.wantCommands {
				if !result.CommandExecuted(cmd) {
					t.Errorf("expected command %q to be executed, but it was not, build output: %s", cmd, result.Output)
				}
			}

			for _, cmd := range tc.doNotWantCommands {
				if result.CommandExecuted(cmd) {
					t.Errorf("expected command %q not to be executed, but it was, build output: %s", cmd, result.Output)
				}
			}
		})
	}
}
