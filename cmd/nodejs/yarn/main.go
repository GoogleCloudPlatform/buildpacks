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

// Implements nodejs/yarn buildpack.
// The npm buildpack installs dependencies using yarn and installs yarn itself if not present.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	cacheTag = "prod dependencies"
	yarnURL  = "https://github.com/yarnpkg/yarn/releases/download/v%[1]s/yarn-v%[1]s.tar.gz"
)

// metadata represents metadata stored for a yarn layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists(nodejs.YarnLock) {
		ctx.OptOut("yarn.lock not found.")
	}
	if !ctx.FileExists("package.json") {
		ctx.OptOut("package.json not found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	if err := installYarn(ctx); err != nil {
		return fmt.Errorf("installing Yarn: %w", err)
	}

	ml := ctx.Layer("yarn")
	nm := filepath.Join(ml.Root, "node_modules")
	ctx.RemoveAll("node_modules")

	nodeEnv := nodejs.NodeEnv()
	cached, meta, err := nodejs.CheckCache(ctx, ml, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", nodejs.YarnLock))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}

	if cached {
		ctx.CacheHit(cacheTag)
		// Restore cached node_modules.
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"}, gcp.WithUserTimingAttribution)
	} else {
		ctx.CacheMiss(cacheTag)
		// Clear cached node_modules to ensure we don't end up with outdated dependencies.
		ctx.ClearLayer(ml)
	}

	// Always run yarn install to run preinstall/postinstall scripts.
	cmd := []string{"yarn", "install", "--non-interactive"}
	if lf := nodejs.LockfileFlag(ctx); lf != "" {
		cmd = append(cmd, lf)
	}
	ctx.Exec(cmd, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution)

	if !cached {
		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution)
	}

	ctx.WriteMetadata(ml, &meta, layers.Build, layers.Cache)

	el := ctx.Layer("env")
	ctx.PrependPathSharedEnv(el, "PATH", filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	ctx.DefaultSharedEnv(el, "NODE_ENV", nodeEnv)
	ctx.WriteMetadata(el, nil, layers.Launch, layers.Build)

	// Configure the entrypoint for production.
	cmd = []string{"yarn", "run", "start"}

	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess(cmd)
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	devmode.AddFileWatcherProcess(ctx, devmode.Config{
		RunCmd: cmd,
		Ext:    devmode.NodeWatchedExtensions,
	})
	devmode.AddSyncMetadata(ctx, devmode.NodeSyncRules)

	return nil
}

func installYarn(ctx *gcp.Context) error {
	// Skip installation if yarn is already installed.
	if result := ctx.Exec([]string{"bash", "-c", "command -v yarn || true"}); result.Stdout != "" {
		ctx.Debugf("Yarn is already installed, skipping installation.")
		return nil
	}

	// Use semver.io to determine the latest available version of Yarn.
	ctx.Logf("Finding latest stable version of Yarn.")
	result := ctx.Exec([]string{"curl", "--silent", "--get", "http://semver.io/yarn/stable"}, gcp.WithUserAttribution)
	version := result.Stdout
	ctx.Logf("The latest stable version of Yarn is v%s", version)

	yarnLayer := "yarn_install"
	yrl := ctx.Layer(yarnLayer)

	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta metadata
	ctx.ReadMetadata(yrl, &meta)
	if version == meta.Version {
		ctx.CacheHit(yarnLayer)
		ctx.Logf("Yarn cache hit, skipping installation.")
	} else {
		ctx.CacheMiss(yarnLayer)
		ctx.ClearLayer(yrl)

		// Download and install yarn in layer.
		ctx.Logf("Installing Yarn v%s", version)
		archiveURL := fmt.Sprintf(yarnURL, version)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, yrl.Root)
		ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	}

	// Store layer flags and metadata.
	meta.Version = version
	ctx.WriteMetadata(yrl, meta, layers.Build, layers.Cache, layers.Launch)
	ctx.Setenv("PATH", filepath.Join(yrl.Root, "bin")+":"+os.Getenv("PATH"))

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    yarnLayer,
		Version: version,
	})
	return nil
}
