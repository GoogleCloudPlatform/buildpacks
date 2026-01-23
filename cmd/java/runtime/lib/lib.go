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

// Implements java/runtime buildpack.
// The runtime buildpack installs the JDK.
package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	javaLayer = "java"
)

// Map with key as stackId and value as the default feature version for that stack.
// We still need to support Java11 on ubuntu18 for OSS applications.
var defaultFeatureVersion = map[string]string{
	"google":         "11",
	"google.gae.18":  "11",
	"google.18":      "11",
	"google.gae.22":  "21",
	"google.min.22":  "21",
	"google.22":      "21",
	"google.24":      "21",
	"google.24.full": "21",
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("java"); result != nil {
		return result, nil
	}

	files := []string{
		"pom.xml",
		".mvn/extensions.xml",
		"build.gradle",
		"build.gradle.kts",
		"settings.gradle.kts",
		"settings.gradle",
		"META-INF/MANIFEST.MF",
	}
	for _, f := range files {
		exists, err := ctx.FileExists(f)
		if err != nil {
			return nil, err
		}
		if exists {
			return gcp.OptInFileFound(f), nil
		}
	}

	javaFiles, err := ctx.Glob("*.java")
	if err != nil {
		return nil, fmt.Errorf("finding .java files: %w", err)
	}
	if len(javaFiles) > 0 {
		return gcp.OptIn("found .java files"), nil
	}
	jarFiles, err := ctx.Glob("*.jar")
	if err != nil {
		return nil, fmt.Errorf("finding .jar files: %w", err)
	}
	if len(jarFiles) > 0 {
		return gcp.OptIn("found .jar files"), nil
	}
	return gcp.OptOut(fmt.Sprintf("none of the following found: %s, *.java, *.jar", strings.Join(files, ", "))), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	featureVersion := stackToVersion(ctx.StackID())
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		featureVersion = v
		ctx.Logf("Using requested runtime feature version: %s", featureVersion)
	} else {
		ctx.Logf("Using latest Java %s runtime version. You can specify a different version with %s: https://github.com/GoogleCloudPlatform/buildpacks#configuration", featureVersion, env.RuntimeVersion)
	}
	l, err := ctx.Layer(javaLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", javaLayer, err)
	}
	jdkRuntime := runtime.OpenJDK
	// Java 21 should fetch Jdk from Canonical instead of Adoptium.
	if strings.HasPrefix(featureVersion, "21") {
		jdkRuntime = runtime.CanonicalJDK
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, jdkRuntime, featureVersion, l)
	return err
}

type binaryPkg struct {
	Link string `json:"link"`
}

type binary struct {
	BinaryPkg    binaryPkg `json:"package"`
	ImageType    string    `json:"image_type"`
	OS           string    `json:"os"`
	Architecture string    `json:"architecture"`
}

type versionData struct {
	Semver string `json:"semver"`
}

type javaRelease struct {
	VersionData versionData `json:"version_data"`
	Binaries    []binary    `json:"binaries"`
}

// stackToVersion returns the default feature version for the given stack.
func stackToVersion(stackID string) string {
	featureVersion := "21"
	if version, ok := defaultFeatureVersion[stackID]; ok {
		featureVersion = version
	}
	return featureVersion
}

// parseVersionJSON parses a JSON array of version information
func parseVersionJSON(jsonStr string) (javaRelease, error) {
	var releases []javaRelease
	if err := json.Unmarshal([]byte(jsonStr), &releases); err != nil {
		return javaRelease{}, fmt.Errorf("parsing JSON response %q: %v", jsonStr, err)
	}
	if len(releases) == 0 {
		return javaRelease{}, fmt.Errorf("empty list of releases")
	}
	return releases[0], nil
}

// extractRelease returns the version name and archiveURL from a javaRelease.
func extractRelease(release javaRelease) (string, string, error) {
	if len(release.Binaries) == 0 {
		return "", "", fmt.Errorf("no binaries in given release %s", release.VersionData.Semver)
	}

	for _, binary := range release.Binaries {
		if binary.ImageType == "jdk" && binary.OS == "linux" && binary.Architecture == "x64" {
			return release.VersionData.Semver, binary.BinaryPkg.Link, nil
		}
	}

	return "", "", fmt.Errorf("jdk/linux/x64 binary not found in release %s", release.VersionData.Semver)
}
