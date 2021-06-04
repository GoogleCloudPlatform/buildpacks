// Copyright 2021 Google LLC
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

// Implements Java GraalVM Native Image buildpack.
// This buildpack installs the GraalVM compiler into a layer and builds a native image of the Java application.
package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb"
)

var (
	requiresGraalvm = []libcnb.BuildPlanRequire{{Name: "graalvm"}}
	planRequires    = libcnb.BuildPlan{Requires: requiresGraalvm}
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(gcp.WithBuildPlans(planRequires)), nil
}

func buildFn(ctx *gcp.Context) error {
	imagePath, err := createImage(ctx)
	if err != nil {
		return err
	}

	ctx.AddWebProcess([]string{imagePath})
	return nil
}

func createImage(ctx *gcp.Context) (string, error) {
	pom := parsePomFile(ctx)
	if pom == nil {
		return buildDefault(ctx)
	}

	if buildProfile, ok := findNativeBuildProfile(ctx, pom); ok {
		return buildMaven(ctx, buildProfile)
	}

	if springBootPluginDefined(ctx, pom) {
		if image, err := buildSpringBoot(ctx); err != nil {
			return "", err
		} else if image != "" {
			return image, nil
		}
	}

	return buildDefault(ctx)
}

// buildDefault builds a native-image in the basic and non-specialized way that can work on any normal
// Java apps and returns the image path. Currently, only supported is an executable JAR in the context.
func buildDefault(ctx *gcp.Context) (string, error) {
	jar, err := java.ExecutableJar(ctx)
	if err != nil {
		return "", fmt.Errorf("finding executable jar: %w", err)
	}
	return buildCommandLine(ctx, []string{"-jar", jar})
}

// buildCommandLine runs the native-image build via command line and returns the image path.
func buildCommandLine(ctx *gcp.Context, buildArgs []string) (string, error) {
	tempImagePath := filepath.Join(ctx.TempDir("native-image"), "native-app")

	command := []string{
		"native-image", "--no-fallback", "--no-server", "-H:+StaticExecutableWithDynamicLibC",
	}
	// Use a temporary image path because this command may generate extra files
	// (*.o and *.build_artifacts.txt) alongside the binary in the temp dir.
	command = append(append(command, buildArgs...), tempImagePath)

	ctx.Exec(command, gcp.WithUserAttribution)

	nativeLayer := ctx.Layer("native-image", gcp.LaunchLayer)
	finalImage := filepath.Join(nativeLayer.Path, "bin", "native-app")

	ctx.MkdirAll(path.Dir(finalImage), 0755)
	ctx.Rename(tempImagePath, finalImage)

	return finalImage, nil
}

// buildMaven runs the Maven native-image build and returns the image path.
func buildMaven(ctx *gcp.Context, buildProfile string) (string, error) {
	command := []string{"mvn", "package", "-DskipTests", "--batch-mode", "-Dhttp.keepAlive=false"}

	if buildProfile != "" {
		command = append(command, "-P"+buildProfile)
	}

	ctx.Exec(command, gcp.WithUserAttribution)

	return findNativeExecutable(ctx)
}

// parsePomFile returns a parsed pom.xml if it exists.
func parsePomFile(ctx *gcp.Context) *java.MavenProject {
	if !ctx.FileExists("pom.xml") {
		return nil
	}

	tmpDir := ctx.TempDir("native-image-maven")

	// Write the effective Maven pom.xml to file.
	effectivePomPath := filepath.Join(tmpDir, "project_effective_pom.xml")
	ctx.Exec([]string{
		"mvn",
		"help:effective-pom",
		"--batch-mode",
		"-Dhttp.keepAlive=false",
		"-Doutput=" + effectivePomPath}, gcp.WithUserAttribution)

	// Parse the effective pom.xml.
	effectivePom := ctx.ReadFile(effectivePomPath)
	project, err := java.ParsePomFile(effectivePom)
	if err != nil {
		ctx.Warnf("A pom.xml was found but unable to be parsed: %v\n", err)
		return nil
	}
	return project
}

// findNativeBuildProfile returns the profile in which the native-image-plugin is defined
// and a bool which returns true if the plugin is found, false if not.
func findNativeBuildProfile(ctx *gcp.Context, project *java.MavenProject) (string, bool) {
	for _, profile := range project.Profiles {
		for _, plugin := range profile.Plugins {
			if plugin.GroupID == "org.graalvm.nativeimage" &&
				plugin.ArtifactID == "native-image-maven-plugin" {
				return profile.ID, true
			}
		}
	}

	ctx.Logf("Did not find a native-image-plugin defined in the pom.xml")
	return "", false
}

// springBootPluginDefined checks if the spring-boot-maven-plugin is defined.
func springBootPluginDefined(ctx *gcp.Context, project *java.MavenProject) bool {
	for _, plugin := range project.Plugins {
		if plugin.GroupID == "org.springframework.boot" &&
			plugin.ArtifactID == "spring-boot-maven-plugin" {
			return true
		}
	}

	ctx.Logf("Did not find a spring-boot-maven-plugin defined in the pom.xml")
	return false
}

// findNativeExecutable returns the path to the executable from the target/ folder
// and only succeeds if exactly 1 is found; returns error otherwise.
func findNativeExecutable(ctx *gcp.Context) (string, error) {
	var allExecutables []string

	targetDir := ctx.ReadDir("target")

	for _, info := range targetDir {
		// If any of the last three bits of the file mode are set, it is executable.
		if !info.IsDir() && info.Mode()&0111 != 0 {
			allExecutables = append(allExecutables, filepath.Join("target", info.Name()))
		}
	}

	if len(allExecutables) != 1 {
		return "", gcp.UserErrorf("expected project to produce exactly 1 executable in target/, but found: %v", allExecutables)
	}

	return allExecutables[0], nil
}

// buildSpringBoot attempts to build a native image from a Spring Boot fat JAR and returns the image path.
// It may return empty if, for example, no Spring Boot fat JAR is found.
func buildSpringBoot(ctx *gcp.Context) (string, error) {
	classpath, main, err := classpathAndMainFromSpringBoot(ctx)
	if err != nil {
		return "", err
	} else if classpath == "" || main == "" {
		return "", nil
	}
	return buildCommandLine(ctx, []string{"--class-path", classpath, main})
}

// classpathAndMainFromSpringBoot returns classpath and main class of an exploded Spring Boot fat JAR
// that is suitable for the application exeuction on a JVM. It may return empty strings if, for example,
// no Spring Boot fat JAR is found.
func classpathAndMainFromSpringBoot(ctx *gcp.Context) (string, string, error) {
	jar, err := java.ExecutableJar(ctx)
	if err != nil {
		ctx.Warnf("Spring Boot project assumed but no main executable JAR found: %v\n", err)
		return "", "", nil
	}
	startClass, err := java.FindManifestValueFromJar(jar, "Start-Class")
	if err != nil {
		return "", "", fmt.Errorf("fetching manifest value from JAR: %q", jar)
	}
	if startClass == "" {
		ctx.Warnf("Spring Boot project assumed but Start-Class undefined in executable JAR: %q", jar)
		return "", "", nil
	}

	explodedJarDir := ctx.TempDir("exploded-jar")
	ctx.Exec([]string{"unzip", "-q", jar, "-d", explodedJarDir}, gcp.WithUserAttribution)

	classes := filepath.Join(explodedJarDir, "BOOT-INF", "classes")
	// TODO(chanseok): using '*' gives a different dependency order than the one computed by Maven.
	// If a Spring Boot fat JAR contain classpath.idx, use it for the exact classpath.
	// https://docs.spring.io/spring-boot/docs/current/reference/html/deployment.html#deployment.containers
	libs := filepath.Join(explodedJarDir, "BOOT-INF", "lib", "*")
	classpath := strings.Join([]string{explodedJarDir, classes, libs}, string(filepath.ListSeparator))

	return classpath, startClass, nil
}
