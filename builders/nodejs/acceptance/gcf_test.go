// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package acceptance

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	npm  = "google.nodejs.npm"
	pnpm = "google.nodejs.pnpm"
	yarn = "google.nodejs.yarn"
)

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "function without package",
			App:  "no_package",
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name: "function without package and with yarn",
			App:  "no_package_yarn",
		},
		{
			Name: "declarative http function",
			App:  "declarative_http",
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"2.1.1\",\"injected\":false}",
			},
		},
		{
			Name:                "declarative CloudEvent function",
			App:                 "declarative_cloud_event",
			RequestType:         acceptance.CloudEventType,
			MustMatchStatusCode: http.StatusNoContent,
		},
		{
			Name:            "function without framework",
			App:             "no_framework",
			MustUse:         []string{npm},
			MustNotUse:      []string{yarn},
			EnableCacheTest: true,
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name:       "function without framework and with yarn",
			App:        "no_framework_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "function with framework",
			App:        "with_framework",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"1.3.2\",\"injected\":false}",
			},
		},
		{
			Name:       "function with framework and with yarn",
			App:        "with_framework_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:            "function with dependencies",
			App:             "with_dependencies",
			MustUse:         []string{npm},
			MustNotUse:      []string{yarn},
			EnableCacheTest: true,
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name:       "function with dependencies and with yarn",
			App:        "with_dependencies_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name:       "function with local dependency",
			App:        "local_dependency",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name:       "function with local dependency and with yarn",
			App:        "local_dependency_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"3.2.0\",\"injected\":true}",
			},
		},
		{
			Name:       "function with gcp-build",
			App:        "with_gcp_build",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with gcp-build and with yarn",
			App:        "with_gcp_build_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "function with runtime env var",
			App:        "with_env_var",
			RunEnv:     []string{"FOO=foo"},
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name:       "function with prepare and with yarn",
			App:        "with_prepare_yarn",
			MustUse:    []string{yarn},
			MustNotUse: []string{npm},
		},
		{
			Name:       "framework loads user dependencies",
			App:        "load_user_dependencies",
			MustUse:    []string{npm},
			MustNotUse: []string{yarn},
		},
		{
			Name: "ESM function",
			// The app uses the "ES Module" which requires nodejs@13
			VersionInclusionConstraint: ">= 13.0.0",
			App:                        "using_esm",
			MustUse:                    []string{npm},
			MustNotUse:                 []string{yarn},
		},
		{
			Name: "Yarn 2 PnP function",
			// The app uses Workers also called 'worker_threads' which are supported with nodejs@12+
			VersionInclusionConstraint: ">= 12.0.0",
			App:                        "yarn_two_pnp",
			MustUse:                    []string{yarn},
			MustNotUse:                 []string{npm},
			Labels: map[string]string{
				"google.functions-framework-version": "{\"runtime\":\"nodejs\",\"version\":\"yarn\",\"injected\":false}",
			},
		},
		{
			Name:       "function with pnpm and typescript",
			App:        "pnpm_typescript",
			MustUse:    []string{pnpm},
			MustNotUse: []string{npm, yarn},
		},
	}

	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := applyStaticAcceptanceTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}

func applyStaticAcceptanceTestOptions(tc acceptance.Test) acceptance.Test {
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	tc.Path = "/testFunction"
	tc.FilesMustExist = append(tc.FilesMustExist,
		"/layers/google.utils.archive-source/src/source-code.tar.gz",
		"/workspace/.googlebuild/source-code.tar.gz",
	)
	return tc
}

func TestFailures(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name:      "syntax error",
			App:       "fail_syntax_error",
			MustMatch: "SyntaxError:",
		},
		{
			Name:      "wrong main",
			App:       "fail_wrong_main",
			MustMatch: "function.js does not exist",
		},
		{
			Name:      "pnpm without framework",
			App:       "no_framework",
			Setup:     addPNPMLock,
			MustMatch: "This project is using pnpm",
		},
		{
			Name:      "function without framework or injection",
			App:       "no_framework",
			Env:       []string{"GOOGLE_SKIP_FRAMEWORK_INJECTION=True"},
			MustMatch: "skipping automatic framework injection has been enabled",
		},
	}

	for _, tc := range testCases {
		tc := applyStaticFailureTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestBuildFailure(t, imageCtx, tc)
		})
	}
}

func applyStaticFailureTestOptions(tc acceptance.FailureTest) acceptance.FailureTest {
	tc.Env = append(tc.Env,
		"GOOGLE_FUNCTION_TARGET=testFunction",
		"X_GOOGLE_TARGET_PLATFORM=gcf",
	)
	return tc
}

// addPNPMLock adds an empty pnpm lock file to the test project
func addPNPMLock(setupCtx acceptance.SetupContext) error {
	fp := filepath.Join(setupCtx.SrcDir, "pnpm-lock.yaml")
	_, err := os.Create(fp)
	return err
}
