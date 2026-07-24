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
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name:  "no files",
			files: map[string]string{},
			want:  100,
		},
		{
			name:  "no files with target_platform",
			files: map[string]string{},
			env:   []string{"X_GOOGLE_TARGET_PLATFORM=gae"},
			want:  0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, DetectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestValidateAppEngineAPIsShouldWarn(t *testing.T) {
	testCases := []struct {
		name                       string
		appEngineAPIsEnvVarEnabled bool
		allDepsResult              string
		directDepsResult           string
		expectedLogOutput          string
	}{
		{
			name:             "App with SDK dependencies and un-enabled APIs should warn",
			allDepsResult:    "",
			directDepsResult: "google.golang.org/appengine",
			expectedLogOutput: "WARNING: There is a dependency on App Engine APIs, but they are " +
				"not enabled in your app.yaml. Set the app_engine_apis property.",
		},
		{
			name:             "Indirect use of SDK libaries should warn",
			allDepsResult:    "google.golang.org/appengine",
			directDepsResult: "",
			expectedLogOutput: "WARNING: There is an indirect dependency on App Engine APIs, " +
				"but they are not enabled in your app.yaml. You may see runtime errors trying to " +
				"access these APIs. Set the app_engine_apis property.",
		},
		{
			name:                       "GAE_APP_ENGINE_APIS but no API use should warn",
			appEngineAPIsEnvVarEnabled: true,
			allDepsResult:              "",
			directDepsResult:           "",
			expectedLogOutput: "WARNING: App Engine APIs are enabled, but don't appear to be " +
				"used, causing a possible performance penalty. Delete app_engine_apis from your app.yaml.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if err := os.Unsetenv("GAE_APP_ENGINE_APIS"); err != nil {
					t.Fatalf("Unexpected error restoring environment: %v", err)
				}
			})
			if tc.appEngineAPIsEnvVarEnabled {
				os.Setenv("GAE_APP_ENGINE_APIS", "TRUE")
			}
			buf := new(bytes.Buffer)
			logger := log.New(buf, "", 0)
			ctx := gcpbuildpack.NewContext(
				gcpbuildpack.WithLogger(logger),
				gcpbuildpack.WithExecCmd(depsMockExecCmd(t, tc.allDepsResult, tc.directDepsResult)),
			)
			if err := validateAppEngineAPIs(ctx); err != nil {
				t.Fatalf("Unexpected error calling validateAppEngineAPIs(ctx) = %v", err)
			}
			logOutput := buf.String()
			if !strings.Contains(logOutput, tc.expectedLogOutput) {
				t.Errorf("validateAppEngineAPIs(ctx)'s log output doesn't contain expected string: got \n%q\n want %q",
					logOutput, tc.expectedLogOutput)
			}
		})
	}
}

// depsMockExecCmd enables controlling the results of directDeps(...) and allDeps(...).
func depsMockExecCmd(t *testing.T, allDepsResult, directDepsResult string) func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		if len(args) != 5 {
			t.Fatalf("Unexpected arguments size: %v", args)
		}
		joinArg := args[3]
		var output string
		if strings.Contains(joinArg, ".Deps") {
			output = allDepsResult
		} else if strings.Contains(joinArg, ".Imports") {
			output = directDepsResult
		} else {
			t.Fatalf("Unexpected argument: %v", joinArg)
		}
		return exec.Command("echo", output)
	}
}
