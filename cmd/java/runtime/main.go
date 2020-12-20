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

// Implements java/runtime buildpack.
// The runtime buildpack installs the JDK.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	javaLayer             = "java"
	javaVersionURL        = "https://api.adoptopenjdk.net/v3/assets/feature_releases/%s/ga?architecture=x64&heap_size=normal&image_type=jdk&jvm_impl=hotspot&os=linux&page=0&page_size=1&project=jdk&sort_order=DESC&vendor=adoptopenjdk"
	defaultFeatureVersion = "11"
	versionKey            = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "java"); result != nil {
		return result, nil
	}

	files := []string{
		"pom.xml",
		".mvn/extensions.xml",
		"build.gradle",
		"build.gradle.kts",
		"project.clj",
		"META-INF/MANIFEST.MF",
	}
	for _, f := range files {
		if ctx.FileExists(f) {
			return gcp.OptInFileFound(f), nil
		}
	}

	if len(ctx.Glob("*.java")) > 0 {
		return gcp.OptIn("found .java files"), nil
	}
	if len(ctx.Glob("*.jar")) > 0 {
		return gcp.OptIn("found .jar files"), nil
	}
	return gcp.OptOut(fmt.Sprintf("none of the following found: %s, *.java, *.jar", strings.Join(files, ", "))), nil
}

func buildFn(ctx *gcp.Context) error {
	featureVersion := defaultFeatureVersion
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		featureVersion = v
		ctx.Logf("Using requested runtime feature version: %s", featureVersion)
	} else {
		ctx.Logf("Using latest Java %s runtime version. You can specify a different version with %s: https://github.com/GoogleCloudPlatform/buildpacks#configuration", defaultFeatureVersion, env.RuntimeVersion)
	}

	releaseURL := fmt.Sprintf(javaVersionURL, featureVersion)
	if code := ctx.HTTPStatus(releaseURL); code != http.StatusOK {
		return gcp.UserErrorf("Java feature version %s does not exist at %s (status %d). You can specify the feature version with %s. See available feature runtime versions at https://api.adoptopenjdk.net/v3/info/available_releases", featureVersion, releaseURL, code, env.RuntimeVersion)
	}

	result := ctx.Exec([]string{"curl", "--fail", "--show-error", "--silent", "--location", releaseURL}, gcp.WithUserAttribution)
	release, err := parseVersionJSON(result.Stdout)
	if err != nil {
		return fmt.Errorf("parsing JSON returned by %s: %w", releaseURL, err)
	}

	version, archiveURL, err := extractRelease(release)
	if err != nil {
		return fmt.Errorf("extracting release returned by %s: %w", releaseURL, err)
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	l := ctx.Layer(javaLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	metaVersion := ctx.GetMetadata(l, versionKey)
	if version == metaVersion {
		ctx.CacheHit(javaLayer)
		return nil
	}
	ctx.CacheMiss(javaLayer)
	ctx.ClearLayer(l)

	// Download and install Java in layer.
	ctx.Logf("Installing Java v%s", version)

	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, l.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	ctx.SetMetadata(l, versionKey, version)
	ctx.AddBuildpackPlanEntry(libcnb.BuildpackPlanEntry{
		Name:     javaLayer,
		Metadata: map[string]interface{}{"version": version},
	})
	return nil
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
