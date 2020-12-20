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
	"flag"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builder, cleanup := acceptance.CreateBuilder(t)
	t.Cleanup(cleanup)

	// In general we fail if we see the string WARNING, because it should be possible to have a project that produces no warnings.
	// In a few cases that is hard and we omit the check.
	testCases := []acceptance.Test{
		{
			Name:          "custom entrypoint",
			App:           "custom_entrypoint",
			Env:           []string{"GOOGLE_ENTRYPOINT=java Main.java"},
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "single jar",
			App:           "single_jar",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "Java11 compat web app",
			App:           "java11_compat_webapp",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "hello quarkus maven",
			App:           "hello_quarkus_maven",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "hello springboot maven",
			App:           "springboot-helloworld",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "hello sparkjava maven",
			App:           "sparkjava-helloworld",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name: "hello micronaut maven",
			App:  "micronaut-helloworld",
			// We don't check for WARNING, because we get a bunch of them from maven-shade-plugin that would be hard to eliminate.
		},
		{
			Name:          "hello vertx maven",
			App:           "vertx-helloworld",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "http server",
			App:           "http-server",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name: "Ktor Kotlin maven mwnw",
			App:  "ktordemo",
			Env:  []string{"GOOGLE_ENTRYPOINT=java -jar target/ktor-0.0.1-jar-with-dependencies.jar"},
			// We don't check for WARNING, because our project-artifact-generated code produces several of them.
		},
		{
			Name:          "gradle micronaut",
			App:           "gradle_micronaut",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "gradlew micronaut",
			App:           "gradlew_micronaut",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "gradle kotlin",
			App:           "gradle-kotlin",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:              "hello quarkus maven with source clearing",
			App:               "hello_quarkus_maven",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true"},
			MustNotOutput:     []string{"WARNING"},
			FilesMustNotExist: []string{"/workspace/src/main/java/hello/Hello.java", "/workspace/pom.xml"},
		},
		{
			Name:              "Gradle with source clearing",
			App:               "gradle_micronaut",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true", "GOOGLE_ENTRYPOINT=java -jar build/libs/helloworld-0.1-all.jar"},
			MustNotOutput:     []string{"WARNING"},
			FilesMustNotExist: []string{"/workspace/src/main/java/example/Application.java", "/workspace/build.gradle"},
		},
		{
			Name:          "Java gradle quarkus",
			App:           "gradle_quarkus",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "leiningen",
			App:           "leiningen",
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:              "Leiningen with source clearing",
			App:               "leiningen",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true", "GOOGLE_ENTRYPOINT=java -jar target/demo-0.0.1-SNAPSHOT-standalone.jar"},
			MustNotOutput:     []string{"WARNING"},
			FilesMustNotExist: []string{"/workspace/src/lein_source/server.clj", "/workspace/project.clj"},
		},
	}
	for _, tc := range testCases {
		tc := tc
		tc.FlakyBuildAttempts = 3

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=java11")

			acceptance.TestApp(t, builder, tc)
		})
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	cleanup := acceptance.UnarchiveTestData()
	// We can't use defer cleanup() here because os.Exit prevents deferred functions from running.
	status := m.Run()
	cleanup()
	os.Exit(status)
}
