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

package acceptance_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

var goVersions = []string{
	"1.11",
	"1.12",
	"1.13",
	"1.14",
	"1.15",
	"1.16",
}

func applyRuntimeVersion(t *testing.T, testCase acceptance.Test, version string) acceptance.Test {
	t.Helper()
	envRuntimeVersion := "GOOGLE_RUNTIME_VERSION"
	for _, e := range testCase.Env {
		if strings.HasPrefix(e, envRuntimeVersion) {
			t.Fatalf("Environment for test %q already contains %q", testCase.Name, envRuntimeVersion)
		}
	}
	testCase.Name = fmt.Sprintf("%s: %s", version, testCase.Name)
	testCase.Env = append(testCase.Env, fmt.Sprintf("%s=%s", envRuntimeVersion, version))
	if version == "1.11" {
		testCase.Env = append(testCase.Env, "GOPROXY=https://proxy.golang.org")
	}
	return testCase
}
