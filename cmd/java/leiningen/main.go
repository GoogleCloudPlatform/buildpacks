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

// Implements java/leiningen buildpack.
// The leiningen buildpack builds Leiningen applications.
package main

import (
	"fmt"
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
	// TODO(b/151198698): Automate Leiningen version updates.
	leiningenVersion = "2.9.5"
	leiningenURL     = "https://raw.githubusercontent.com/technomancy/leiningen/%s/bin/lein"
	leiningenLayer   = "leiningen"
	m2Layer          = "m2"
	versionKey       = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.FileExists("project.clj") {
		return gcp.OptInFileFound("project.clj"), nil
	}
	return gcp.OptOut("project.clj was not found"), nil
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

	var lein string
	if ctx.FileExists("lein") {
		lein = "./lein"
	} else if leiningenInstalled(ctx) {
		lein = "lein"
	} else {
		lein, err = installLeiningen(ctx)
		if err != nil {
			return fmt.Errorf("installing Leiningen: %w", err)
		}
	}

	command := []string{lein, "uberjar"}

	if buildArgs := os.Getenv(env.BuildArgs); buildArgs != "" {
		if strings.Contains(buildArgs, "maven.repo.local") {
			ctx.Warnf("Detected maven.repo.local property set in GOOGLE_BUILD_ARGS. Maven caching may not work properly.")
		}
		command = append(command, buildArgs)
	}

	ctx.Exec(command, gcp.WithStdoutTail, gcp.WithUserAttribution)

	// Store the build steps in a script to be run on each file change.
	if devmode.Enabled(ctx) {
		devmode.WriteBuildScript(ctx, m2CachedRepo.Path, "~/.m2", command)
	}

	return nil
}

func leiningenInstalled(ctx *gcp.Context) bool {
	result := ctx.Exec([]string{"bash", "-c", "command -v lein || true"})
	return result.Stdout != ""
}

func installLeiningen(ctx *gcp.Context) (string, error) {
	leinl := ctx.Layer(leiningenLayer, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(leinl, versionKey)
	if leiningenVersion == metaVersion {
		ctx.CacheHit(leiningenLayer)
		ctx.Logf("Maven cache hit, skipping installation.")
		return filepath.Join(leinl.Path, "lein"), nil
	}
	ctx.CacheMiss(leiningenLayer)
	ctx.ClearLayer(leinl)

	// Download and install leiningen in layer.
	ctx.Logf("Installing Leiningen v%s", leiningenVersion)
	archiveURL := fmt.Sprintf(leiningenURL, leiningenVersion)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return "", gcp.UserErrorf("Leiningen version %s does not exist at %s (status %d).", leiningenVersion, archiveURL, code)
	}
	lein := filepath.Join(leinl.Path, "lein")
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s --output %s", archiveURL, lein)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	os.Chmod(lein, 0755)

	ctx.SetMetadata(leinl, versionKey, leiningenVersion)
	return lein, nil
}
