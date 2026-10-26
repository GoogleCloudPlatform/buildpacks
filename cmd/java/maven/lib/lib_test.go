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
	"os"
	"path/filepath"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "pom.xml",
			files: map[string]string{
				"pom.xml": "",
			},
			want: 0,
		},
		{
			name: ".mvn/extensions.xml",
			files: map[string]string{
				".mvn/extensions.xml": "",
			},
			want: 0,
		},
		{
			name:  "no pom.xml",
			files: map[string]string{},
			want:  100,
		},
		{
			name: "use GOOGLE_BUILDABLE",
			files: map[string]string{
				"testmodule/pom.xml": "",
			},
			env:  []string{"GOOGLE_BUILDABLE=testmodule"},
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		name              string
		app               string
		envs              []string
		opts              []buildpacktest.Option
		mocks             []*mockprocess.Mock
		wantCommands      []string
		doNotWantCommands []string
		files             map[string]string
	}{
		{
			name: "maven build argument",
			app:  "hello_quarkus_maven",
			mocks: []*mockprocess.Mock{
				mockprocess.New(`^bash -c command -v mvn || true`, mockprocess.WithStdout("Apache Maven")),
			},
			envs: []string{fmt.Sprintf("%s=clean package", java.MavenBuildArgs)},
			wantCommands: []string{
				"mvn clean package",
			},
			doNotWantCommands: []string{
				"mvn clean package --batch-mode -DskipTests -Dhttp.keepAlive=false",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []buildpacktest.Option{
				buildpacktest.WithTestName(tc.name),
				buildpacktest.WithApp(tc.app),
				buildpacktest.WithEnvs(tc.envs...),
				buildpacktest.WithExecMocks(tc.mocks...),
			}

			opts = append(opts, tc.opts...)
			result, err := buildpacktest.RunBuild(t, BuildFn, opts...)
			if err != nil {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
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

func TestBuildPreservesHomeM2WhenInstallerDisabled(t *testing.T) {
	homeDir := t.TempDir()
	m2Dir := filepath.Join(homeDir, ".m2")
	if err := os.MkdirAll(m2Dir, 0755); err != nil {
		t.Fatalf("creating %s: %v", m2Dir, err)
	}
	settingsFile := filepath.Join(m2Dir, "settings.xml")
	wantContent := "<settings/>"
	if err := os.WriteFile(settingsFile, []byte(wantContent), 0644); err != nil {
		t.Fatalf("writing %s: %v", settingsFile, err)
	}

	opts := []buildpacktest.Option{
		buildpacktest.WithApp("hello_quarkus_maven"),
		buildpacktest.WithEnvs("HOME=" + homeDir),
		buildpacktest.WithDisabledCapabilities(java.MavenInstallerCapability),
		buildpacktest.WithExecMocks(
			mockprocess.New(`^bash -c command -v mvn || true`, mockprocess.WithStdout("Apache Maven")),
		),
	}
	result, err := buildpacktest.RunBuild(t, BuildFn, opts...)
	if err != nil {
		t.Fatalf("error running build: %v, logs: %s", err, result.Output)
	}

	fi, err := os.Lstat(m2Dir)
	if err != nil {
		t.Fatalf("Lstat(%s) failed: %v", m2Dir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected %s to remain a real directory, but it is a symlink", m2Dir)
	}

	gotContent, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("expected %s to be preserved, but ReadFile failed: %v", settingsFile, err)
	}
	if string(gotContent) != wantContent {
		t.Errorf("settings.xml content = %q, want %q", string(gotContent), wantContent)
	}
}
