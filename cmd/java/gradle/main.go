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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	gradleVersionURL = "https://services.gradle.org/versions/current"
	gradleLayer      = "gradle"
	m2Layer          = "m2"
)

// gradleMetadata represents metadata stored for a gradle layer.
type gradleMetadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("build.gradle") && !ctx.FileExists("build.gradle.kts") {
		ctx.OptOut("Neither build.gradle nor build.gradle.kts found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	var repoMeta java.RepoMetadata
	m2CachedRepo := ctx.Layer(m2Layer)
	ctx.ReadMetadata(m2CachedRepo, &repoMeta)
	java.CheckCacheExpiration(ctx, &repoMeta, m2CachedRepo)

	var gradle string
	var err error
	if ctx.FileExists("gradlew") {
		gradle = "./gradlew"
	} else if gradleInstalled(ctx) {
		gradle = "gradle"
	} else {
		gradle, err = installGradle(ctx)
		if err != nil {
			return fmt.Errorf("installing Gradle: %w", err)
		}
	}

	command := []string{gradle, "assemble", "-x", "test", "--project-cache-dir=" + m2CachedRepo.Root}

	if buildArgs := os.Getenv(env.BuildArgs); buildArgs != "" {
		if strings.Contains(buildArgs, "project-cache-dir") {
			ctx.Warnf("Detected project-cache-dir property set in GOOGLE_BUILD_ARGS. Dependency caching may not work properly.")
		}
		command = append(command, buildArgs)
	}

	if !ctx.Debug() {
		command = append(command, "--quiet")
	}

	ctx.ExecUser(command)

	ctx.WriteMetadata(m2CachedRepo, &repoMeta, layers.Cache)

	return nil
}

func gradleInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v gradle || true"})
	return result.Stdout != ""
}

type gradleVersion struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadUrl"`
}

// installGradle installs Gradle and returns the path of the gradle binary
func installGradle(ctx *gcp.Context) (string, error) {
	gradlel := ctx.Layer(gradleLayer)

	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta gradleMetadata
	ctx.ReadMetadata(gradlel, &meta)

	version, downloadURL, err := fetchGradleVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching latest Gradle version: %w", err)
	}

	if version == meta.Version {
		ctx.CacheHit(gradleLayer)
		ctx.Logf("Gradle cache hit, skipping installation.")
		return filepath.Join(gradlel.Root, "bin", "gradle"), nil
	}
	ctx.CacheMiss(gradleLayer)
	ctx.ClearLayer(gradlel)

	// Download and install gradle in layer.
	ctx.Logf("Installing Gradle v%s", version)
	if code := ctx.HTTPStatus(downloadURL); code != http.StatusOK {
		return "", fmt.Errorf("Gradle version %s does not exist at %s (status %d)", version, downloadURL, code)
	}

	tmpDir := "/tmp"
	gradleZip := filepath.Join(tmpDir, "gradle.zip")
	defer ctx.RemoveAll(gradleZip)

	curl := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s --output %s", downloadURL, gradleZip)
	ctx.Exec([]string{"bash", "-c", curl})

	unzip := fmt.Sprintf("unzip -q %s -d %s", gradleZip, tmpDir)
	ctx.Exec([]string{"bash", "-c", unzip})

	gradleExtracted := filepath.Join(tmpDir, fmt.Sprintf("gradle-%s", version))
	defer ctx.RemoveAll(gradleExtracted)
	install := fmt.Sprintf("mv %s/* %s", gradleExtracted, gradlel.Root)
	ctx.Exec([]string{"bash", "-c", install})

	meta.Version = version
	ctx.WriteMetadata(gradlel, meta, layers.Cache)
	return filepath.Join(gradlel.Root, "bin", "gradle"), nil
}

// fetchGradleVersion returns the latest gradle version, its downloadURL, and an error in that order.
func fetchGradleVersion(ctx *gcp.Context) (string, string, error) {
	if code := ctx.HTTPStatus(gradleVersionURL); code != http.StatusOK {
		return "", "", fmt.Errorf("Gradle latest version info does not exist at %s (status %d)", gradleVersionURL, code)
	}

	jsonStr := ctx.Exec([]string{"curl", "--silent", gradleVersionURL}).Stdout
	var gv gradleVersion
	if err := json.Unmarshal([]byte(jsonStr), &gv); err != nil {
		return "", "", fmt.Errorf("parsing JSON response from URL %q: %v", gradleVersionURL, err)
	}
	return gv.Version, gv.DownloadURL, nil
}
