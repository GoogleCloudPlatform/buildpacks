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
package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

const (
	// Buildpack identifiers used to verify that buildpacks were or were not used.
	entrypoint      = "google.config.entrypoint"
	javaClearSource = "google.java.clear-source"
	javaEntrypoint  = "google.java.entrypoint"
	javaExplodedJar = "google.java.exploded-jar"
	javaGradle      = "google.java.gradle"
	javaMaven       = "google.java.maven"
	javaRuntime     = "google.java.runtime"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptanceJava(t *testing.T) {
	imageContext, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name: "simple Java 11 application",
			App:  "simple",
			Env: []string{
				"GOOGLE_ENTRYPOINT=java Main.java",
				"GOOGLE_RUNTIME_VERSION=11",
			},
			MustUse:         []string{javaRuntime, entrypoint},
			MustNotUse:      []string{javaEntrypoint},
			EnableCacheTest: true,
		},
		{
			Name: "simple Java 17 application",
			App:  "simple",
			Env: []string{
				"GOOGLE_ENTRYPOINT=java Main.java",
				"GOOGLE_RUNTIME_VERSION=17",
			},
			MustUse:         []string{javaRuntime, entrypoint},
			MustNotUse:      []string{javaEntrypoint},
			EnableCacheTest: true,
		},
		{
			Name: "Java runtime version respected",
			App:  "simple",
			// Checking runtime version 8 to ensure that it is not downloading latest Java 11 version.
			Path: "/version?want=8",
			Env: []string{
				"GOOGLE_ENTRYPOINT=javac Main.java; java Main",
				"GOOGLE_RUNTIME_VERSION=8",
			},
			MustUse:    []string{javaRuntime, entrypoint},
			MustNotUse: []string{javaEntrypoint},
		},
		{
			Name:            "Java maven",
			App:             "hello_quarkus_maven",
			MustUse:         []string{javaMaven, javaRuntime, javaEntrypoint},
			MustNotUse:      []string{entrypoint},
			EnableCacheTest: true,
		},
		{
			Name:                "Java maven (Dev Mode)",
			App:                 "hello_quarkus_maven",
			Env:                 []string{"GOOGLE_DEVMODE=1"},
			FilesMustExist:      []string{"/layers/google.java.maven/m2", "/layers/google.java.maven/m2/bin/.devmode_rebuild.sh"},
			MustRebuildOnChange: "/workspace/src/main/java/hello/Hello.java",
		},
		{
			Name:       "Java gradle",
			App:        "gradle_micronaut",
			Env:        []string{"GOOGLE_ENTRYPOINT=java -jar build/libs/helloworld-0.1-all.jar"},
			MustUse:    []string{javaGradle, javaRuntime, entrypoint},
			MustNotUse: []string{javaEntrypoint},
		},
		{
			Name:       "Java mono repo",
			App:        "gradle_mono_repo",
			MustUse:    []string{javaGradle, javaRuntime, javaEntrypoint},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:                "Java gradle (Dev Mode)",
			App:                 "gradle_micronaut",
			Env:                 []string{"GOOGLE_DEVMODE=1"},
			FilesMustExist:      []string{"/layers/google.java.gradle/cache", "/layers/google.java.gradle/cache/bin/.devmode_rebuild.sh"},
			MustRebuildOnChange: "/workspace/src/main/java/example/HelloController.java",
		},
		{
			Name: "polyglot maven java 11",
			App:  "polyglot-maven",
			Env: []string{
				"GOOGLE_RUNTIME_VERSION=11",
			},
			MustUse:    []string{javaMaven, javaRuntime, javaEntrypoint},
			MustNotUse: []string{entrypoint},
		},
		{
			Name: "polyglot maven java 17",
			App:  "polyglot-maven",
			Env: []string{
				"GOOGLE_RUNTIME_VERSION=17",
			},
			MustUse:    []string{javaMaven, javaRuntime, javaEntrypoint},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "Maven build args",
			App:        "maven_testing_profile",
			Env:        []string{"GOOGLE_BUILD_ARGS=--settings=maven_settings.xml -Dnative=false"},
			MustUse:    []string{javaMaven, javaRuntime, javaEntrypoint},
			MustNotUse: []string{entrypoint},
		},
		{
			Name:       "Gradle build args",
			App:        "gradle_test_env",
			Env:        []string{"GOOGLE_BUILD_ARGS=-Denv=test", "GOOGLE_ENTRYPOINT=java -jar build/libs/helloworld-0.1-all.jar"},
			MustUse:    []string{javaGradle, javaRuntime, entrypoint},
			MustNotUse: []string{javaEntrypoint},
		},
		{
			Name: "Exploded Jar java 11",
			App:  "exploded_jar",
			Env: []string{
				"GOOGLE_RUNTIME_VERSION=11",
			},
			MustUse: []string{javaRuntime, javaExplodedJar},
		},
		{
			Name: "Exploded Jar java 17",
			App:  "exploded_jar",
			Env: []string{
				"GOOGLE_RUNTIME_VERSION=17",
			},
			MustUse: []string{javaRuntime, javaExplodedJar},
		},
		{
			Name:              "Maven with source clearing",
			App:               "hello_quarkus_maven",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true"},
			MustUse:           []string{javaMaven, javaRuntime, javaEntrypoint, javaClearSource},
			MustNotUse:        []string{entrypoint},
			FilesMustExist:    []string{"/workspace/target/hello-1-runner.jar"},
			FilesMustNotExist: []string{"/workspace/src/main/java/hello/Hello.java", "/workspace/pom.xml"},
		},
		{
			Name:              "Gradle with source clearing",
			App:               "gradle_micronaut",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true", "GOOGLE_ENTRYPOINT=java -jar build/libs/helloworld-0.1-all.jar"},
			MustUse:           []string{javaGradle, javaRuntime, entrypoint, javaClearSource},
			MustNotUse:        []string{javaEntrypoint},
			FilesMustExist:    []string{"/workspace/build/libs/helloworld-0.1-all.jar"},
			FilesMustNotExist: []string{"/workspace/src/main/java/example/Application.java", "/workspace/build.gradle"},
		},
		{
			Name:       "Multi-module",
			App:        "multi_module",
			Env:        []string{"GOOGLE_BUILDABLE=hello"},
			MustUse:    []string{javaMaven, javaRuntime, javaEntrypoint},
			MustNotUse: []string{entrypoint},
		},
	}
	for _, tc := range testCases {
		tc := tc
		tc.FlakyBuildAttempts = 3

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, imageContext, tc)
		})
	}
}
