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

// Implements /bin/build for java/maven buildpack.
package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	dateFormat = time.RFC3339Nano
	// repoExpiration is an arbitrary amount of time of 10 weeks to refresh the cache layer.
	// TODO: Investigate proper cache-clearing strategy.
	repoExpiration = time.Duration(time.Hour * 24 * 7 * 10)
	// TODO: Automate Maven version updates.
	mavenVersion = "3.6.3"
	mavenURL     = "https://downloads.apache.org/maven/maven-3/%[1]s/binaries/apache-maven-%[1]s-bin.tar.gz"
	mavenLayer   = "maven"
	m2Layer      = "m2"
)

// mavenMetadata represents metadata stored for a maven layer.
type mavenMetadata struct {
	Version string `toml:"version"`
}

type repoMetadata struct {
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("pom.xml") {
		ctx.OptOut("pom.xml not found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	var repoMeta repoMetadata
	m2CachedRepo := ctx.Layer(m2Layer)
	ctx.ReadMetadata(m2CachedRepo, &repoMeta)
	checkCacheExpiration(ctx, &repoMeta, m2CachedRepo)

	var command string
	var err error
	if ctx.FileExists("mvnw") {
		command = "./mvnw"
	} else if mvnInstalled(ctx) {
		command = "mvn"
	} else {
		command, err = installMaven(ctx)
		if err != nil {
			return fmt.Errorf("installing Maven: %w", err)
		}
	}

	ctx.ExecUser([]string{command, "clean", "package", "--batch-mode", "-DskipTests", "-Dmaven.repo.local=" + m2CachedRepo.Root})

	ctx.WriteMetadata(m2CachedRepo, &repoMeta, layers.Cache)

	return nil
}

// checkCacheExpiration clears the m2 layer and sets a new expiry timestamp when the cache is past expiration.
func checkCacheExpiration(ctx *gcp.Context, repoMeta *repoMetadata, m2CachedRepo *layers.Layer) {
	future := time.Now().Add(repoExpiration).Format(dateFormat)

	if repoMeta.ExpiryTimestamp == "" {
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}

	t, err := time.Parse(dateFormat, repoMeta.ExpiryTimestamp)
	if err != nil {
		ctx.Debugf("Could not parse date %q, resetting expiration: %v", repoMeta.ExpiryTimestamp, err)
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}

	if t.Before(time.Now()) {
		// Clear the local maven repo after some fixed amount of time so that it doesn't continually grow.
		ctx.ClearLayer(m2CachedRepo)
		repoMeta.ExpiryTimestamp = future
		return
	}
	return
}

func mvnInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v mvn || true"})
	return result.Stdout != ""
}

// installMaven installs Maven and returns the path of the mvn binary
func installMaven(ctx *gcp.Context) (string, error) {
	mvnl := ctx.Layer(mavenLayer)

	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta mavenMetadata
	ctx.ReadMetadata(mvnl, &meta)

	if mavenVersion == meta.Version {
		ctx.CacheHit(mavenLayer)
		ctx.Logf("Maven cache hit, skipping installation.")
		return filepath.Join(mvnl.Root, "bin", "mvn"), nil
	}
	ctx.CacheMiss(mavenLayer)
	ctx.ClearLayer(mvnl)

	// Download and install maven in layer.
	ctx.Logf("Installing Maven v%s", mavenVersion)
	archiveURL := fmt.Sprintf(mavenURL, mavenVersion)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return "", gcp.UserErrorf("Maven version %s does not exist at %s (status %d).", mavenVersion, archiveURL, code)
	}
	command := fmt.Sprintf("curl --fail --show-error --silent --location %s | tar xz --directory=%s --strip-components=1", archiveURL, mvnl.Root)
	ctx.Exec([]string{"bash", "-c", command})

	meta.Version = mavenVersion
	ctx.WriteMetadata(mvnl, meta, layers.Cache)
	return filepath.Join(mvnl.Root, "bin", "mvn"), nil
}
