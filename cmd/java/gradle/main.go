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

// Implements java/gradle buildpack.
// The gradle buildpack builds Gradle applications.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

const (
	gradleVersion   = "6.5.1"
	gradleDistroURL = "https://services.gradle.org/distributions/gradle-%s-bin.zip"
	gradleLayer     = "gradle"
	cacheLayer      = "cache"
	versionKey      = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.FileExists("build.gradle") {
		return gcp.OptInFileFound("build.gradle"), nil
	}
	if ctx.FileExists("build.gradle.kts") {
		return gcp.OptInFileFound("build.gradle.kts"), nil
	}
	return gcp.OptOut("neither build.gradle nor build.gradle.kts found"), nil
}

func buildFn(ctx *gcp.Context) error {
	gradleCachedRepo := ctx.Layer(cacheLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)

	java.CheckCacheExpiration(ctx, gradleCachedRepo)

	homeGradle := filepath.Join(ctx.HomeDir(), ".gradle")
	// Symlink the gradle-cache layer into ~/.gradle. If ~/.gradle already exists, delete it first.
	// If it exists as a symlink, RemoveAll will remove the link, not anything it's linked to.
	ctx.RemoveAll(homeGradle)
	ctx.Symlink(gradleCachedRepo.Path, homeGradle)

	var gradle string
	if ctx.FileExists("gradlew") {
		gradle = "./gradlew"
	} else if gradleInstalled(ctx) {
		gradle = "gradle"
	} else {
		var err error
		gradle, err = installGradle(ctx)
		if err != nil {
			return fmt.Errorf("installing Gradle: %w", err)
		}
	}

	command := []string{gradle, "clean", "assemble", "-x", "test", "--build-cache"}

	if buildArgs := os.Getenv(env.BuildArgs); buildArgs != "" {
		if strings.Contains(buildArgs, "project-cache-dir") {
			ctx.Warnf("Detected project-cache-dir property set in GOOGLE_BUILD_ARGS. Dependency caching may not work properly.")
		}
		command = append(command, buildArgs)
	}

	if !ctx.Debug() && !devmode.Enabled(ctx) {
		command = append(command, "--quiet")
	}

	ctx.Exec(command, gcp.WithUserAttribution)

	// Store the build steps in a script to be run on each file change.
	if devmode.Enabled(ctx) {
		devmode.WriteBuildScript(ctx, gradleCachedRepo.Path, "~/.gradle", command)
	}

	return nil
}

func gradleInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v gradle || true"})
	return result.Stdout != ""
}

// installGradle installs Gradle and returns the path of the gradle binary
func installGradle(ctx *gcp.Context) (string, error) {
	gradlel := ctx.Layer(gradleLayer, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)

	metaVersion := ctx.GetMetadata(gradlel, versionKey)
	// Check the metadata in the cache layer to determine if we need to proceed.
	if gradleVersion == metaVersion {
		ctx.CacheHit(gradleLayer)
		ctx.Logf("Gradle cache hit, skipping installation.")
		return filepath.Join(gradlel.Path, "bin", "gradle"), nil
	}
	ctx.CacheMiss(gradleLayer)
	ctx.ClearLayer(gradlel)

	downloadURL := fmt.Sprintf(gradleDistroURL, gradleVersion)
	// Download and install gradle in layer.
	ctx.Logf("Installing Gradle v%s", gradleVersion)
	if code := ctx.HTTPStatus(downloadURL); code != http.StatusOK {
		return "", fmt.Errorf("Gradle version %s does not exist at %s (status %d)", gradleVersion, downloadURL, code)
	}

	tmpDir := "/tmp"
	gradleZip := filepath.Join(tmpDir, "gradle.zip")
	defer ctx.RemoveAll(gradleZip)

	curl := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s --output %s", downloadURL, gradleZip)
	ctx.Exec([]string{"bash", "-c", curl}, gcp.WithUserAttribution)

	unzip := fmt.Sprintf("unzip -q %s -d %s", gradleZip, tmpDir)
	ctx.Exec([]string{"bash", "-c", unzip}, gcp.WithUserAttribution)

	gradleExtracted := filepath.Join(tmpDir, fmt.Sprintf("gradle-%s", gradleVersion))
	defer ctx.RemoveAll(gradleExtracted)
	install := fmt.Sprintf("mv %s/* %s", gradleExtracted, gradlel.Path)
	ctx.Exec([]string{"bash", "-c", install}, gcp.WithUserTimingAttribution)

	ctx.SetMetadata(gradlel, versionKey, gradleVersion)
	return filepath.Join(gradlel.Path, "bin", "gradle"), nil
}
