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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
)

func init() {
	acceptance.DefineFlags()
}

var goVersions = []string{
	"1.11",
	"1.12",
	"1.13",
	"1.14",
	"1.15",
	"1.16",
}

func applyRuntimeVersionTest(t *testing.T, testCase acceptance.Test, version string) acceptance.Test {
	t.Helper()
	testCase.Env = applyRuntimeVersionEnv(t, testCase.Env, version)
	testCase.Name = fmt.Sprintf("%s: %s", version, testCase.Name)
	return testCase
}

func applyRuntimeVersionFailureTest(t *testing.T, testCase acceptance.FailureTest, version string) acceptance.FailureTest {
	t.Helper()
	testCase.Env = applyRuntimeVersionEnv(t, testCase.Env, version)
	testCase.Name = fmt.Sprintf("%s: %s", version, testCase.Name)
	return testCase
}

func applyRuntimeVersionEnv(t *testing.T, environment []string, version string) []string {
	t.Helper()
	envMap := envToMap(t, environment)
	addEnvVar(t, envMap, "GOOGLE_RUNTIME_VERSION", version)
	// X_GOOGLE_TARGET_PLATFORM defined tells us that a build contacted RCS. GOOGLE_RUNTIME will only have
	// a value when a build uses RCS for configuration, so only GCF and GAE. For cloud run builds,
	// GOOGLE_RUNTIME will not have a value.
	if hasEnvVar(env.XGoogleTargetPlatform, envMap) {
		addEnvVar(t, envMap, "GOOGLE_RUNTIME", fmt.Sprintf("go%s", strings.ReplaceAll(version, ".", "")))
	}
	if version == "1.11" {
		addEnvVar(t, envMap, "GOPROXY", "https://proxy.golang.org")
	}
	return mapToEnv(envMap)
}

func addEnvVar(t *testing.T, env map[string]string, name, value string) {
	t.Helper()
	if hasEnvVar(name, env) {
		t.Fatalf("Environment already contains %q: %s", name, env)
	}
	env[name] = value
}

func hasEnvVar(name string, env map[string]string) bool {
	_, ok := env[name]
	return ok
}

func envToMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	results := make(map[string]string, len(env))
	for _, e := range env {
		if e == "" {
			t.Fatalf("unexpected empty value for environment variable")
		}
		splits := strings.SplitN(e, "=", 2)
		if len(splits) > 2 {
			t.Fatalf("environment variable %q has unexpected syntax: want 'var=value'", e)
		}
		// an environment variable isn't required to have a value, it might just be defined
		value := ""
		if len(splits) == 2 {
			value = splits[1]
		}
		results[splits[0]] = value
	}
	return results
}

func mapToEnv(env map[string]string) []string {
	results := make([]string, 0, len(env))
	for n, v := range env {
		results = append(results, formatEnvVar(n, v))
	}
	return results
}

func formatEnvVar(name, value string) string {
	if value == "" {
		return name
	}
	return fmt.Sprintf("%s=%s", name, value)
}
