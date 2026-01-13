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
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

// updateGradleVersions replaces various dependency versions in the application source that are
// incompatible with Gradle 7.
func updateGradleVersions(setupCtx acceptance.SetupContext) error {
	err := replaceInFile(filepath.Join(setupCtx.SrcDir, "build.gradle"), map[string]string{
		"5.2.0": "7.1.2", // com.github.johnrengelman.shadow plugin
	})
	if err != nil {
		return err
	}

	err = replaceInFile(filepath.Join(setupCtx.SrcDir, "build.gradle.kts"), map[string]string{
		"4.0.4":   "7.1.2",          // com.github.johnrengelman.shadow plugin
		"1.3.21":  "1.6.21",         // kotlin jvm and kapt plugins
		"compile": "implementation", // `compile` was deprecated in favor of `implementation`
	})
	if err != nil {
		return err
	}

	gwFp := filepath.Join(setupCtx.SrcDir, "gradle/wrapper/gradle-wrapper.properties")
	err = replaceInFile(gwFp, map[string]string{
		// gradle distro versions:
		"5.3.1": "7.4.2",
		"6.1":   "7.4.2",
		"6.4.1": "7.4.2",
	})
	return err
}

// replaceInFile is a helper to find and replace strings in a file at a given path.
func replaceInFile(path string, repacements map[string]string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for k, v := range repacements {
		content = bytes.Replace(content, []byte(k), []byte(v), -1)
	}
	os.Remove(path)
	return os.WriteFile(path, content, 0644)
}

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	// In general we fail if we see the string WARNING, because it should be possible to have a project that produces no warnings.
	// In a few cases that is hard and we omit the check.
	testCases := []acceptance.Test{
		{
			Name:          "custom_entrypoint",
			App:           "custom_entrypoint",
			Env:           []string{"GOOGLE_ENTRYPOINT=java Main.java"},
			MustNotOutput: []string{"WARNING"},
		},
		{
			Name:          "single_jar",
			App:           "single_jar",
			MustNotOutput: []string{"WARNING"},
		},
		{
			VersionInclusionConstraint: ">=11.0.0 <12.0.0",
			Name:                       "Java11_compat_web_app",
			App:                        "java11_compat_webapp",
			MustNotOutput:              []string{"WARNING"},
		},
		{
			VersionInclusionConstraint: ">=17.0.0 <18.0.0",
			Name:                       "Java17_compat_web_app",
			App:                        "java17_compat_webapp",
		},
		{
			Name: "hello quarkus maven",
			App:  "hello_quarkus_maven",
		},
		{
			Name:          "hello_springboot_maven",
			App:           "springboot-helloworld",
			MustNotOutput: []string{"ERROR"},
		},
		{
			Name:            "hello_sparkjava_maven",
			App:             "sparkjava-helloworld",
			MustNotOutput:   []string{"ERROR"},
			EnableCacheTest: true,
		},
		{
			Name: "hello_micronaut_maven",
			App:  "micronaut-helloworld",
			// We don't check for WARNING, because we get a bunch of them from maven-shade-plugin that would be hard to eliminate.
		},
		{
			Name:          "hello_vertx_maven",
			App:           "vertx-helloworld",
			MustNotOutput: []string{"ERROR"},
		},
		{
			Name:            "http_server",
			App:             "http-server",
			MustNotUse:      []string{"google.java.spring-boot"},
			MustNotOutput:   []string{"ERROR"},
			EnableCacheTest: true,
		},
		{
			Name:                       "Ktor_Kotlin_maven_mwnw",
			App:                        "ktordemo",
			VersionInclusionConstraint: "<25.0.0",
			Env:                        []string{"GOOGLE_ENTRYPOINT=java -jar target/ktor-0.0.1-jar-with-dependencies.jar"},
			// We don't check for WARNING, because our project-artifact-generated code produces several of them.
		},
		{
			Name:              "hello_quarkus_maven_with_source_clearing",
			App:               "hello_quarkus_maven",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true"},
			FilesMustNotExist: []string{"/workspace/src/main/java/hello/Hello.java", "/workspace/pom.xml"},
		},
		{
			Name:          "gradle_micronaut",
			App:           "gradle_micronaut",
			MustNotOutput: []string{"WARNING"},
			Setup:         updateGradleVersions,
		},
		{
			Name:                       "gradlew_micronaut",
			App:                        "gradlew_micronaut",
			VersionInclusionConstraint: "<25.0.0",
			MustNotOutput:              []string{"WARNING"},
			Setup:                      updateGradleVersions,
		},
		{
			Name:                       "gradle_kotlin",
			App:                        "gradle-kotlin",
			VersionInclusionConstraint: "<25.0.0",
			Setup:                      updateGradleVersions,
		},
		{
			Name:              "Gradle_with_source_clearing",
			App:               "gradle_micronaut",
			Env:               []string{"GOOGLE_CLEAR_SOURCE=true", "GOOGLE_ENTRYPOINT=java -jar build/libs/helloworld-0.1-all.jar"},
			MustNotOutput:     []string{"WARNING"},
			FilesMustNotExist: []string{"/workspace/src/main/java/example/Application.java", "/workspace/build.gradle"},
			Setup:             updateGradleVersions,
		},
		{
			Name:                       "Java_gradle_quarkus",
			App:                        "gradle_quarkus",
			VersionInclusionConstraint: "<25.0.0",
			MustNotOutput:              []string{"WARNING"},
			Setup:                      updateGradleVersions,
		},
		// Spring boot app, spring-boot-buildpack must opt in for all java versions >=Java17
		{
			Name:                       "hello_springboot_maven_for_java_17_and_above",
			App:                        "springboot-helloworld",
			VersionInclusionConstraint: ">=17.0.0",
			MustUse:                    []string{"google.java.spring-boot"},
			MustNotOutput:              []string{"ERROR"},
		},
		{
			Name:                       "hello_springboot_maven_for_java_11",
			App:                        "springboot-helloworld",
			VersionInclusionConstraint: "<17.0.0",
			MustNotUse:                 []string{"google.java.spring-boot"},
			MustNotOutput:              []string{"ERROR"},
		},
	}
	for _, tc := range acceptance.FilterTests(t, imageCtx, testCases) {
		tc := tc
		tc.FlakyBuildAttempts = 3

		t.Run(tc.Name, func(t *testing.T) {
			// Running these tests in parallel causes the server to run out of disk space.
			// t.Parallel()

			tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")

			acceptance.TestApp(t, imageCtx, tc)
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
