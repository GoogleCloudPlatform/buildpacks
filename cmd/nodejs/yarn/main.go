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
	cacheTag   = "prod dependencies"
	yarnLayer  = "yarn_engine"
	versionKey = "version"
	yarnURL    = "https://yarnpkg.com/downloads/%[1]s/yarn-v%[1]s.tar.gz"
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
	yarn2, err := nodejs.IsYarn2(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	// Plug'n'Play mode is a Yarn2 feature in which dependencies are always bundled with the application
	// source: https://yarnpkg.com/features/pnp
	pnpMode := yarn2 && nodejs.IsYarnPNP(ctx)

	if err := installYarn(ctx); err != nil {
		return fmt.Errorf("installing Yarn: %w", err)
	}

	ml := ctx.Layer("yarn_modules", gcp.BuildLayer, gcp.CacheLayer)
	updateCache, err := restoreCachedModules(ctx, ml, pnpMode)
	if err != nil {
		return err
	}

	// Always run yarn install to execute customer's lifecycle hooks.
	cmd, err := nodejs.YarnInstallCmd(ctx, yarn2, pnpMode)
	if err != nil {
		return err
	}
	ctx.Exec(cmd, gcp.WithUserAttribution)

	if updateCache {
		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		// Update the cache before we purge to ensure it includes the devDependencies.
		ctx.Exec([]string{"cp", "--archive", "node_modules", filepath.Join(ml.Path, "node_modules")}, gcp.WithUserTimingAttribution)
	}

	// Run the gcp-build script if it exists.
	if err := gcpBuild(ctx); err != nil {
		return err
	}

	nodeEnv := nodejs.NodeEnv()
	switch {
	case nodeEnv != nodejs.EnvProduction:
		ctx.Logf("Retaining devDependencies because NODE_ENV=%q", nodeEnv)
	case yarn2 && !nodejs.HasYarnWorkspacePlugin(ctx):
		ctx.Warnf("Keeping devDependencies because the Yarn workspace-tools plugin is not installed. You can add it to your project by running 'yarn plugin import workspace-tools'")
	case yarn2:
		// For Yarn2, dependency pruning is via the workspaces plugin.
		ctx.Logf("Pruning devDependencies")
		ctx.Exec([]string{"yarn", "workspaces", "focus", "--all", "--production"}, gcp.WithUserAttribution)
	default:
		// For Yarn1, setting `--production=true` causes all `devDependencies` to be deleted.
		ctx.Logf("Pruning devDependencies")
		ctx.Exec([]string{"yarn", "install", "--frozen-lockfile", "--ignore-scripts", "--prefer-offline", "--production=true"}, gcp.WithUserAttribution)
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
	version, err := nodejs.DetectYarnVersion(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	yrl := ctx.Layer(yarnLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(yrl, versionKey)
	if version == metaVersion {
		ctx.CacheHit(yarnLayer)
		ctx.Logf("Yarn cache hit, skipping installation.")
	} else {
		ctx.CacheMiss(yarnLayer)
		ctx.ClearLayer(yrl)
		// Download and install yarn in layer.
		ctx.Logf("Installing Yarn v%s", version)
		archiveURL := fmt.Sprintf(yarnURL, version)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, yrl.Path)
		ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(yrl, versionKey, version)
	// We need to update the path here to ensure the version we just installed take precendence over
	// anything pre-installed in the base image.
	ctx.Setenv("PATH", filepath.Join(yrl.Path, "bin")+":"+os.Getenv("PATH"))
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     yarnLayer,
		Metadata: map[string]interface{}{"version": version},
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

// restoreCachedModules copies the cached node_modules from the provided layer into the application
// dir if they can be re-used and returns true if cache needs to be updated.
func restoreCachedModules(ctx *gcp.Context, ml *libcnb.Layer, pnpMode bool) (bool, error) {
	if pnpMode {
		// In Yarn2 Plug'n'Play mode all modules are included with the application source so there
		// is no point in adding them to a cache layer.
		ctx.ClearLayer(ml)
		return false, nil
	}
	nm := filepath.Join(ml.Path, "node_modules")
	ctx.RemoveAll("node_modules")

	cached, err := nodejs.CheckCache(ctx, ml, cache.WithFiles("package.json", nodejs.YarnLock))
	if err != nil {
		return true, fmt.Errorf("checking cache: %w", err)
	}

	if cached {
		// The yarn.lock hasn't been updated since we last built so the cached node_modules should be
		// up-to-date. There is no need to update the layer cache after we run "yarn install" because
		// it should be a no-op.
		ctx.CacheHit(cacheTag)
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"}, gcp.WithUserTimingAttribution)
		return false, nil
	}

	// The dependencies listed in the yarn.lock file have changed. Clear the layer cache and update
	// it after we run yarn install
	ctx.CacheMiss(cacheTag)
	ctx.ClearLayer(ml)
	return true, nil
}
