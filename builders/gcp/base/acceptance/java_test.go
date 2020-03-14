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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "simple Java application",
			App:     "java/simple",
			Env:     []string{"GOOGLE_ENTRYPOINT=java Main.java"},
			MustUse: []string{javaRuntime, javaEntrypoint},
		},
		{
			Name: "Java runtime version respected",
			App:  "java/simple",
			// Checking runtime version 11.0.5+10 to ensure that it is not downloading latest version.
			Path: "/version?want=11.0.5+10",
			Env: []string{
				"GOOGLE_ENTRYPOINT=java Main.java",
				"GOOGLE_RUNTIME_VERSION=11.0.5+10",
			},
			MustUse: []string{javaRuntime, javaEntrypoint},
		},
		{
			Name:       "Java selected via GOOGLE_RUNTIME",
			App:        "override",
			Env:        []string{"GOOGLE_RUNTIME=java", "GOOGLE_ENTRYPOINT=java Main.java"},
			MustUse:    []string{javaRuntime, javaEntrypoint},
			MustNotUse: []string{goRuntime, nodeRuntime, pythonRuntime},
		},
		{
			Name:    "Java maven",
			App:     "java/hello_quarkus_maven",
			MustUse: []string{javaMaven, javaRuntime, javaEntrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builder, tc)
		})
	}
}
