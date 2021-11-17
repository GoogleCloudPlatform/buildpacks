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
	"os/user"
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
	if ctx.FileExists("pom.xml") {
		return gcp.OptInFileFound("pom.xml"), nil
	}
	if ctx.FileExists(".mvn/extensions.xml") {
		return gcp.OptInFileFound(".mvn/extensions.xml"), nil
	}
	return gcp.OptOut("none of the following found: pom.xml or .mvn/extensions.xml."), nil
}

func buildFn(ctx *gcp.Context) error {
	m2CachedRepo := ctx.Layer(m2Layer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	java.CheckCacheExpiration(ctx, m2CachedRepo)

	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting current user: %v", err)
	}
	// TODO(b/157297290): just use os.Getenv("HOME") when that is consistent with /etc/passwd.
	homeM2 := filepath.Join(usr.HomeDir, ".m2")
	// Symlink the m2 layer into ~/.m2. If ~/.m2 already exists, delete it first.
	// If it exists as a symlink, RemoveAll will remove the link, not anything it's linked to.
	// We can't just use `-Dmaven.repo.local`. It does set the path to `m2/repo` but it fails
	// to set the path to `m2/wrapper` which is used by mvnw to download Maven.
	ctx.RemoveAll(homeM2)
	ctx.Symlink(m2CachedRepo.Path, homeM2)

	addJvmConfig(ctx)

	var mvn string
	if ctx.FileExists("mvnw") {
		// With CRLF endings, the "\r" gets seen as part of the shebang target, which doesn't exist.
		ensureUnixLineEndings(ctx, "mvnw")
		mvn = "./mvnw"
	} else if mvnInstalled(ctx) {
		mvn = "mvn"
	} else {
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
func addJvmConfig(ctx *gcp.Context) {
	version := os.Getenv(env.RuntimeVersion)
	if version == "8" || strings.HasPrefix(version, "8.") {
		// We don't need this workaround on Java 8, and in fact it fails there because there's no --add-opens option.
		return
	}
	configFile := ".mvn/jvm.config"
	if ctx.FileExists(configFile) {
		return
	}
	if err := os.MkdirAll(".mvn", 0755); err != nil {
		ctx.Logf("Could not create .mvn, reflection warnings may not be disabled: %v", err)
		return
	}
	jvmOptions := "--add-opens java.base/java.lang=ALL-UNNAMED"
	if err := ioutil.WriteFile(configFile, []byte(jvmOptions), 0644); err != nil {
		ctx.Logf("Could not create %s, reflection warnings may not be disabled: %v", configFile, err)
	}
}

func mvnInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v mvn || true"})
	return result.Stdout != ""
}

// installMaven installs Maven and returns the path of the mvn binary
func installMaven(ctx *gcp.Context) (string, error) {
	mvnl := ctx.Layer(mavenLayer, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)

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
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return "", gcp.UserErrorf("Maven version %s does not exist at %s (status %d).", mavenVersion, archiveURL, code)
	}
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, mvnl.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	ctx.SetMetadata(mvnl, versionKey, mavenVersion)
	return filepath.Join(mvnl.Path, "bin", "mvn"), nil
}

// Replace CRLF with LF
func ensureUnixLineEndings(ctx *gcp.Context, file ...string) {

	if !ctx.IsWritable(file...) {
		return
	}

	path := filepath.Join(file...)
	data := ctx.ReadFile(path)

	data = bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})

	ctx.WriteFile(path, data, 0755)
}
