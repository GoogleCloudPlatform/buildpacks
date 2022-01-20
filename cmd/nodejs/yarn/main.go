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
	"github.com/buildpacks/libcnb"
)

const (
	cacheTag    = "prod dependencies"
	yarnVersion = "1.22.5"
	yarnURL     = "https://github.com/yarnpkg/yarn/releases/download/v%[1]s/yarn-v%[1]s.tar.gz"
	versionKey  = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.FileExists(nodejs.YarnLock) {
		return gcp.OptOutFileNotFound("yarn.lock"), nil
	}
	if !ctx.FileExists("package.json") {
		return gcp.OptOutFileNotFound("package.json"), nil
	}

	return gcp.OptIn("found yarn.lock and package.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := installYarn(ctx); err != nil {
		return fmt.Errorf("installing Yarn: %w", err)
	}

	ml := ctx.Layer("yarn_modules", gcp.BuildLayer, gcp.CacheLayer)
	nm := filepath.Join(ml.Path, "node_modules")
	ctx.RemoveAll("node_modules")

	nodeEnv := nodejs.NodeEnv()
	cached, err := nodejs.CheckCache(ctx, ml, cache.WithFiles("package.json", nodejs.YarnLock))
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

	// Always run yarn install to execute customer's lifecycle hooks. Setting --production=false
	// causes the devDependencies to be installed regardless of the NODE_ENV value, so any hooks
	// have access to them. We purge the devDependencies from the final app image below.
	cmd := []string{"yarn", "install", "--non-interactive", "--prefer-offline", "--production=false"}
	lf, err := nodejs.LockfileFlag(ctx)
	if err != nil {
		return err
	}
	if lf != "" {
		cmd = append(cmd, lf)
	}
	ctx.Exec(cmd, gcp.WithUserAttribution)

	if !cached {
		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		// Update the cache before we purge to ensure it includes the devDependencies.
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution)
	}

	// Run the gcp-build script if it exists.
	if err := gcpBuild(ctx); err != nil {
		return err
	}

	if nodeEnv == nodejs.EnvProduction {
		ctx.Logf("Pruning devDependencies.")
		// Setting `--production=true` causes all `devDependencies` to be deleted.
		ctx.Exec([]string{"yarn", "install", "--frozen-lockfile", "--ignore-scripts", "--prefer-offline", "--production=true"}, gcp.WithUserAttribution)
	} else {
		ctx.Logf("Retaining devDependencies because NODE_ENV=%q", nodeEnv)
	}

	el := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodeEnv)

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

	yarnLayer := "yarn_install"
	yrl := ctx.Layer(yarnLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(yrl, versionKey)
	if yarnVersion == metaVersion {
		ctx.CacheHit(yarnLayer)
		ctx.Logf("Yarn cache hit, skipping installation.")
	} else {
		ctx.CacheMiss(yarnLayer)
		ctx.ClearLayer(yrl)

		// Download and install yarn in layer.
		ctx.Logf("Installing Yarn v%s", yarnVersion)
		archiveURL := fmt.Sprintf(yarnURL, yarnVersion)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, yrl.Path)
		ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(yrl, versionKey, yarnVersion)
	ctx.Setenv("PATH", filepath.Join(yrl.Path, "bin")+":"+os.Getenv("PATH"))
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     yarnLayer,
		Metadata: map[string]interface{}{"version": yarnVersion},
		Launch:   true,
		Build:    true,
	})
	return nil
}

func gcpBuild(ctx *gcp.Context) error {
	p, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if p.Scripts.GCPBuild != "" {
		ctx.Exec([]string{"yarn", "run", "gcp-build"}, gcp.WithUserAttribution)
	}
	return nil
}
