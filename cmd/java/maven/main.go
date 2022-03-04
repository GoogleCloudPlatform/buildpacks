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

// Implements java/maven buildpack.
// The maven buildpack builds Maven applications.
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	// TODO(b/151198698): Automate Maven version updates.
	mavenVersion = "3.6.3"
	mavenURL     = "https://downloads.apache.org/maven/maven-3/%[1]s/binaries/apache-maven-%[1]s-bin.tar.gz"
	mavenLayer   = "maven"
	m2Layer      = "m2"
	versionKey   = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	pomExists, err := ctx.FileExists("pom.xml")
	if err != nil {
		return nil, err
	}
	if pomExists {
		return gcp.OptInFileFound("pom.xml"), nil
	}
	extXMLExists, err := ctx.FileExists(".mvn/extensions.xml")
	if err != nil {
		return nil, err
	}
	if extXMLExists {
		return gcp.OptInFileFound(".mvn/extensions.xml"), nil
	}
	return gcp.OptOut("none of the following found: pom.xml or .mvn/extensions.xml."), nil
}

func buildFn(ctx *gcp.Context) error {
	m2CachedRepo, err := ctx.Layer(m2Layer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", m2Layer, err)
	}
	java.CheckCacheExpiration(ctx, m2CachedRepo)

	homeM2 := filepath.Join(ctx.HomeDir(), ".m2")
	// Symlink the m2 layer into ~/.m2. If ~/.m2 already exists, delete it first.
	// If it exists as a symlink, RemoveAll will remove the link, not anything it's linked to.
	// We can't just use `-Dmaven.repo.local`. It does set the path to `m2/repo` but it fails
	// to set the path to `m2/wrapper` which is used by mvnw to download Maven.
	if err := ctx.RemoveAll(homeM2); err != nil {
		return err
	}
	if err := ctx.Symlink(m2CachedRepo.Path, homeM2); err != nil {
		return err
	}

	if err := addJvmConfig(ctx); err != nil {
		return err
	}

	var mvn string
	mvnwExists, err := ctx.FileExists("mvnw")
	if err != nil {
		return err
	}
	if mvnwExists {
		// With CRLF endings, the "\r" gets seen as part of the shebang target, which doesn't exist.
		if err := ensureUnixLineEndings(ctx, "mvnw"); err != nil {
			return fmt.Errorf("ensuring unix newline characters: %w", err)
		}
		mvn = "./mvnw"
	} else if mvnInstalled(ctx) {
		mvn = "mvn"
	} else {
		var err error
		mvn, err = installMaven(ctx)
		if err != nil {
			return fmt.Errorf("installing Maven: %w", err)
		}
	}

	command := []string{mvn, "clean", "package", "--batch-mode", "-DskipTests", "-Dhttp.keepAlive=false"}

	if buildArgs := os.Getenv(env.BuildArgs); buildArgs != "" {
		if strings.Contains(buildArgs, "maven.repo.local") {
			ctx.Warnf("Detected maven.repo.local property set in GOOGLE_BUILD_ARGS. Maven caching may not work properly.")
		}
		command = append(command, buildArgs)
	}

	if !ctx.Debug() && !devmode.Enabled(ctx) {
		command = append(command, "--quiet")
	}

	ctx.Exec(command, gcp.WithStdoutTail, gcp.WithUserAttribution)

	// Store the build steps in a script to be run on each file change.
	if devmode.Enabled(ctx) {
		devmode.WriteBuildScript(ctx, m2CachedRepo.Path, "~/.m2", command)
	}

	return nil
}

// addJvmConfig is a workaround for https://github.com/google/guice/issues/1133, an "illegal reflective access" warning.
// When that bug has been fixed in a version of mvn we can use, we can remove this workaround.
// Write a JVM flag to .mvn/jvm.config in the project being built to suppress the warning.
// Don't do anything if there already is a .mvn/jvm.config.
func addJvmConfig(ctx *gcp.Context) error {
	version := os.Getenv(env.RuntimeVersion)
	if version == "8" || strings.HasPrefix(version, "8.") {
		// We don't need this workaround on Java 8, and in fact it fails there because there's no --add-opens option.
		return nil
	}
	configFile := ".mvn/jvm.config"
	configFileExists, err := ctx.FileExists(configFile)
	if err != nil {
		return err
	}
	if configFileExists {
		return nil
	}
	if err := os.MkdirAll(".mvn", 0755); err != nil {
		ctx.Logf("Could not create .mvn, reflection warnings may not be disabled: %v", err)
		return nil
	}
	jvmOptions := "--add-opens java.base/java.lang=ALL-UNNAMED"
	if err := ioutil.WriteFile(configFile, []byte(jvmOptions), 0644); err != nil {
		ctx.Logf("Could not create %s, reflection warnings may not be disabled: %v", configFile, err)
	}
	return nil
}

func mvnInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v mvn || true"})
	return result.Stdout != ""
}

// installMaven installs Maven and returns the path of the mvn binary
func installMaven(ctx *gcp.Context) (string, error) {
	mvnl, err := ctx.Layer(mavenLayer, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return "", fmt.Errorf("creating %v layer: %w", mavenLayer, err)
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(mvnl, versionKey)
	if mavenVersion == metaVersion {
		ctx.CacheHit(mavenLayer)
		ctx.Logf("Maven cache hit, skipping installation.")
		return filepath.Join(mvnl.Path, "bin", "mvn"), nil
	}
	ctx.CacheMiss(mavenLayer)
	ctx.ClearLayer(mvnl)

	// Download and install maven in layer.
	ctx.Logf("Installing Maven v%s", mavenVersion)
	archiveURL := fmt.Sprintf(mavenURL, mavenVersion)
	code, err := ctx.HTTPStatus(archiveURL)
	if err != nil {
		return "", err
	}
	if code != http.StatusOK {
		return "", gcp.UserErrorf("Maven version %s does not exist at %s (status %d).", mavenVersion, archiveURL, code)
	}
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, mvnl.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	ctx.SetMetadata(mvnl, versionKey, mavenVersion)
	return filepath.Join(mvnl.Path, "bin", "mvn"), nil
}

// Replace CRLF with LF
func ensureUnixLineEndings(ctx *gcp.Context, file ...string) error {
	isWriteable, err := ctx.IsWritable(file...)
	if err != nil {
		return err
	}
	if !isWriteable {
		return nil
	}

	path := filepath.Join(file...)
	data, err := ctx.ReadFile(path)
	if err != nil {
		return err
	}

	data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})

	if err := ctx.WriteFile(path, data, os.FileMode(0755)); err != nil {
		return err
	}
	return nil
}
