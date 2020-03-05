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

// TODO: Add an equivalent script to the open-source version.
// Run these tests using:
//   apphosting/runtime/titanium/buildpacks/tools/run-acceptance-tests.sh --runtime=gaejava11
//
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	cleanup := acceptance.UnarchiveTestData(t)
	t.Cleanup(cleanup)

	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "custom entrypoint",
			App:  "custom_entrypoint",
			Env:  []string{"GOOGLE_ENTRYPOINT=java Main.java"},
		},
		{
			Name: "single jar",
			App:  "single_jar",
		},
		{
			Name: "hello quarkus maven",
			App:  "hello_quarkus_maven",
		},
		{
			Name: "Ktor Kotlin maven mwnw",
			App:  "ktordemo",
			Env:  []string{"GOOGLE_ENTRYPOINT=java -jar target/ktor-0.0.1-jar-with-dependencies.jar"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=java11")

			acceptance.TestApp(t, builder, tc)
		})
	}
}
