// Copyright 2025 Google LLC
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

// Package lib implements the java/spring-boot buildpack.
package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/Masterminds/semver"
)

const (
	springBootVersionManifest = "Spring-Boot-Version"
)

// DetectFn detects if the application is a Spring Boot application.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	runtimeVersion := os.Getenv("GOOGLE_RUNTIME_VERSION")
	if runtimeVersion != "" {
		cleanedVersionStr := strings.Replace(runtimeVersion, "_", "-", 1)
		v, err := semver.NewVersion(cleanedVersionStr)
		if err != nil {
			return gcp.OptOut(fmt.Sprintf("Failed to parse runtime version '%s' as semver: %v", runtimeVersion, err)), nil
		}
		if v.Major() < 17 {
			return gcp.OptOut(fmt.Sprintf("Runtime version %s is less than java17.", runtimeVersion)), nil
		}
	}
	ctx.Logf("Checking for packaged JAR")
	springBootVersion, _ := SpringBootVersionInManifest(ctx)
	if springBootVersion != "" {
		ctx.Logf("Detected Spring Boot version %s in manifest", springBootVersion)
		return gcp.OptIn("Opted in, Spring Boot version found in manifest."), nil
	}
	ctx.Logf("Checking for Spring Boot in pom.xml. JAR file not present at detect time.")
	if exists, _ := ctx.FileExists("pom.xml"); exists {
		project, err := parsePomFile(ctx)
		if err != nil {
			ctx.Logf("Failed to parse effective pom: %v", err)
		}
		if err == nil && project != nil && (springBootStarterDefined(ctx, project) || springBootPluginDefined(ctx, project)) {
			ctx.Logf("Detected Spring Boot in pom.xml")
			return gcp.OptIn("Opted in, Spring Boot detected."), nil
		}
	}
	return gcp.OptOut("Not a Spring Boot project"), nil
}

// BuildFn adds to the metric if it is a Spring Boot application.
// Since we just want to record metrics, we won't return an error anywhere
func BuildFn(ctx *gcp.Context) error {
	springBootVersion, _ := SpringBootVersionInManifest(ctx)
	if springBootVersion != "" {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.JavaSpringBootUsageCounterID).Increment(1)
		ctx.Logf("Detected Spring Boot version %s. Incremented counter.", springBootVersion)
	} else {
		ctx.Logf("Not a Spring Boot application (Spring-Boot-Version not found in manifest). Skipping Spring Boot buildpack logic.")
	}
	return nil
}

// SpringBootVersionInManifest returns the Spring Boot version in the manifest of the application JAR.
func SpringBootVersionInManifest(ctx *gcp.Context) (string, error) {
	appJar, err := java.ExecutableJar(ctx)
	if err != nil {
		ctx.Logf("Error finding executable jar: %v", err)
		return "", nil
	}
	if appJar == "" {
		ctx.Logf("No executable JAR found. Skipping Spring Boot buildpack logic.")
		return "", nil
	}
	ctx.Logf("Found potential application JAR: %s", appJar)
	springBootVersion, err := java.FindManifestValueFromJar(appJar, springBootVersionManifest)
	if err != nil {
		ctx.Logf("Error reading manifest from jar %s: %v", appJar, err)
		return "", nil
	}
	return springBootVersion, nil
}

// parsePomFile returns a parsed pom.xml if it exists.
func parsePomFile(ctx *gcp.Context) (*java.MavenProject, error) {
	exists, err := ctx.FileExists("pom.xml")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	pomContent, err := ctx.ReadFile("pom.xml")
	if err != nil {
		return nil, fmt.Errorf("reading pom.xml: %w", err)
	}
	project, err := java.ParsePomFile(pomContent)
	if err != nil {
		ctx.Warnf("Found pom.xml but failed to parse it: %v", err)
		return nil, nil
	}
	return project, nil
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

// springBootStarterDefined checks if the spring-boot-starter is defined.
func springBootStarterDefined(ctx *gcp.Context, project *java.MavenProject) bool {
	for _, dependency := range project.Dependencies {
		if strings.Contains(dependency.ArtifactID, "spring-boot-starter") {
			return true
		}
	}
	ctx.Logf("Did not find a spring-boot-starter defined in the pom.xml")
	return false
}
