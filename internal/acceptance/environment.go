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

package acceptance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

func prepareEnvTest(t *testing.T, test Test) map[string]string {
	env := envToMap(t, test.Env)
	env["GOOGLE_DEBUG"] = "true"
	if shouldApplyRuntimeVersion(env) {
		applyRuntimeVersion(t, env, runtimeVersion)
	}
	return env
}

func prepareEnvFailureTest(t *testing.T, fTest FailureTest) map[string]string {
	env := envToMap(t, fTest.Env)
	env["GOOGLE_DEBUG"] = "true"
	if !fTest.SkipBuilderOutputMatch {
		env["BUILDER_OUTPUT"] = "/tmp/builderoutput"
		env["EXPECTED_BUILDER_OUTPUT"] = fTest.MustMatch
	}
	if shouldApplyRuntimeVersion(env) {
		applyRuntimeVersion(t, env, runtimeVersion)
	}
	return env
}

func shouldApplyRuntimeVersion(environment map[string]string) bool {
	if runtimeVersion == "" {
		return false
	}
	return !hasEnvVar("GOOGLE_RUNTIME_VERSION", environment)
}

func applyRuntimeVersion(t *testing.T, environment map[string]string, version string) {
	t.Helper()
	addEnvVar(t, environment, "GOOGLE_RUNTIME_VERSION", version)
	// X_GOOGLE_TARGET_PLATFORM defined tells us that a build contacted RCS. GOOGLE_RUNTIME will only have
	// a value when a build uses RCS for configuration, so only GCF and GAE. For cloud run builds,
	// GOOGLE_RUNTIME will not have a value.
	if hasEnvVar(env.XGoogleTargetPlatform, environment) && runtimeName != "" {
		runtimeEnvVar, err := runtime.FormatName(runtimeName, version)
		if err != nil {
			t.Fatalf("Error formatting the runtime name for %q: %v", runtimeName, err)
		}
		addEnvVar(t, environment, "GOOGLE_RUNTIME", runtimeEnvVar)

		if environment[env.XGoogleTargetPlatform] == "gae" {
			// X_GOOGLE_SKIP_RUNTIME_LAUNCH tells the buildpacks to skip adding runtime to the launch layer.
			// This is needed for GAE as it uses an overridden run-image which already has the runtime installed.
			addEnvVar(t, environment, "X_GOOGLE_SKIP_RUNTIME_LAUNCH", "true")
		}
	}
	if runtimeName == "go" && strings.HasPrefix(version, "1.11") {
		addEnvVar(t, environment, "GOPROXY", "https://proxy.golang.org")
	}
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
