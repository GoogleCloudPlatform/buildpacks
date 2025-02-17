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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb/v2"
)

const (
	invokerMain = "com.google.cloud.functions.invoker.runner.Invoker"
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
	entrypoint, err := createImage(ctx)
	if err != nil {
		return err
	}

	ctx.AddWebProcess(entrypoint)
	return nil
}

// createImage builds a native-image and returns an image entrypoint. It handles
// all the logic for which workflow to use (e.g native-image build via command
// line or maven profile) based on the project setup.
func createImage(ctx *gcp.Context) ([]string, error) {
	pom, err := parsePomFile(ctx)
	if err != nil {
		return nil, fmt.Errorf("parsing pom file: %w", err)
	}
	if pom == nil {
		return buildDefault(ctx)
	}
	if functionTarget, ok := os.LookupEnv(env.FunctionTarget); ok {
		return buildFunctionsFramework(ctx, functionTarget, pom)
	}

	if buildProfile, ok := findNativeBuildProfile(ctx, pom); ok {
		return buildMaven(ctx, buildProfile)
	}

	// The presence of the `spring-boot-maven-plugin` may not always guarantee that
	// the project will generate a Spring-Boot fat JAR. In the case where a Spring
	// Boot fat JAR is not found, we fall through to the default mode of building a
	// native image for standard Java apps.
	if springBootPluginDefined(ctx, pom) {
		if entrypoint, err := buildSpringBoot(ctx); err != nil {
			return nil, err
		} else if entrypoint != nil {
			return entrypoint, nil
		}
	}

	return buildDefault(ctx)
}

// buildDefault builds a native-image in the basic and non-specialized way that can work on any normal
// Java apps and returns the image entrypoint. Currently, only supported is an executable JAR in the context.
func buildDefault(ctx *gcp.Context) ([]string, error) {
	jar, err := java.ExecutableJar(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding executable jar: %w", err)
	}
	return buildCommandLine(ctx, []string{"-jar", jar})
}

// buildCommandLine runs the native-image build via command line and returns the image entrypoint.
func buildCommandLine(ctx *gcp.Context, buildArgs []string) ([]string, error) {
	niDir, err := ctx.TempDir("native-image")
	if err != nil {
		return nil, err
	}
	tempImagePath := filepath.Join(niDir, "native-app")

	// Use a temporary image path because this command may generate extra files
	// (*.o and *.build_artifacts.txt) alongside the binary in the temp dir.
	userArgs := os.Getenv(env.NativeImageBuildArgs)
	command := fmt.Sprintf("native-image --no-fallback --no-server -H:+StaticExecutableWithDynamicLibC %s %s %s",
		userArgs, strings.Join(buildArgs, " "), tempImagePath)

	if _, err := ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution); err != nil {
		return nil, err
	}

	nativeLayer, err := ctx.Layer("native-image", gcp.LaunchLayer)
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}
	finalImage := filepath.Join(nativeLayer.Path, "bin", "native-app")

	if err := ctx.MkdirAll(finalImage, 0755); err != nil {
		return nil, err
	}
	if err := ctx.Rename(tempImagePath, finalImage); err != nil {
		return nil, err
	}

	return []string{finalImage}, nil
}

// buildMaven runs the Maven native-image build and returns the image entrypoint.
func buildMaven(ctx *gcp.Context, buildProfile string) ([]string, error) {
	mvn, err := java.MvnCmd(ctx)
	if err != nil {
		return nil, err
	}
	command := []string{mvn, "package", "-DskipTests", "--batch-mode", "-Dhttp.keepAlive=false"}

	if buildProfile != "" {
		command = append(command, "-P"+buildProfile)
	}

	if _, err := ctx.Exec(command, gcp.WithUserAttribution); err != nil {
		return nil, err
	}

	imagePath, err := findNativeExecutable(ctx)
	if err != nil {
		return nil, err
	}
	return []string{imagePath}, nil
}

// parsePomFile returns a parsed pom.xml if it exists.
func parsePomFile(ctx *gcp.Context) (*java.MavenProject, error) {
	pomExists, err := ctx.FileExists("pom.xml")
	if err != nil {
		return nil, err
	}
	if !pomExists {
		return nil, nil
	}

	tmpDir, err := ctx.TempDir("native-image-maven")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	// Write the effective Maven pom.xml to file.
	effectivePomPath := filepath.Join(tmpDir, "project_effective_pom.xml")
	mvn, err := java.MvnCmd(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := ctx.Exec([]string{
		mvn,
		"help:effective-pom",
		"--batch-mode",
		"-Dhttp.keepAlive=false",
		"-Doutput=" + effectivePomPath}, gcp.WithUserAttribution); err != nil {
		return nil, err
	}

	// Parse the effective pom.xml.
	effectivePom, err := ctx.ReadFile(effectivePomPath)
	if err != nil {
		return nil, err
	}
	project, err := java.ParsePomFile(effectivePom)
	if err != nil {
		ctx.Warnf("A pom.xml was found but unable to be parsed: %v\n", err)
		return nil, nil
	}
	return project, nil
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

	targetDir, err := ctx.ReadDir("target")
	if err != nil {
		return "", err
	}

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

// buildSpringBoot attempts to build a native image from a Spring Boot fat JAR and returns the image entrypoint.
// It may return empty if, for example, no Spring Boot fat JAR is found.
func buildSpringBoot(ctx *gcp.Context) ([]string, error) {
	classpath, main, err := classpathAndMainFromSpringBoot(ctx)
	if err != nil {
		return nil, err
	} else if classpath == "" || main == "" {
		return nil, nil
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

	explodedJarDir, err := ctx.TempDir("exploded-jar")
	if err != nil {
		return "", "", fmt.Errorf("creating temp directory: %w", err)
	}
	if _, err := ctx.Exec([]string{"unzip", "-q", jar, "-d", explodedJarDir}, gcp.WithUserAttribution); err != nil {
		return "", "", err
	}

	classes := filepath.Join(explodedJarDir, "BOOT-INF", "classes")
	// TODO(chanseok): using '*' gives a different dependency order than the one computed by Maven.
	// If a Spring Boot fat JAR contain classpath.idx, use it for the exact classpath.
	// https://docs.spring.io/spring-boot/docs/current/reference/html/deployment.html#deployment.containers
	libs := filepath.Join(explodedJarDir, "BOOT-INF", "lib", "*")
	classpath := strings.Join([]string{explodedJarDir, classes, libs}, string(filepath.ListSeparator))

	return classpath, startClass, nil
}

// buildFunctionsFramework runs the native-image build for the standard GCF workflow and returns the image entrypoint.
func buildFunctionsFramework(ctx *gcp.Context, functionTarget string, project *java.MavenProject) ([]string, error) {
	classpath, err := createFunctionsClasspath(ctx, project)
	if err != nil {
		return nil, err
	}

	entrypoint, err := buildCommandLine(ctx, []string{"-cp", classpath, invokerMain})
	if err != nil {
		return nil, err
	}
	functionsFrameworkEntrypoint := append(entrypoint, "--target", functionTarget)
	return functionsFrameworkEntrypoint, nil
}

// createFunctionsClasspath generates the full classpath to be used with native-image command line for GCF workflow
func createFunctionsClasspath(ctx *gcp.Context, project *java.MavenProject) (string, error) {
	jarName := fmt.Sprintf("%s-%s.jar", project.ArtifactID, project.Version)
	applicationJar := filepath.Join("target", jarName)
	jarExists, err := ctx.FileExists(applicationJar)
	if err != nil {
		return "", err
	}
	if !jarExists {
		return "", gcp.UserErrorf("finding application JAR: %s", applicationJar)
	}
	dependencies := filepath.Join("target", "dependency", "*")
	classpath := strings.Join([]string{os.Getenv(java.FFJarPathEnv), applicationJar, dependencies}, string(filepath.ListSeparator))

	return classpath, nil
}
