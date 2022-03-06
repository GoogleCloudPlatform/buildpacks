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
	yarnLockExists, err := ctx.FileExists(nodejs.YarnLock)
	if err != nil {
		return nil, err
	}
	if !yarnLockExists {
		return gcp.OptOutFileNotFound("yarn.lock"), nil
	}
	pkgJSONExists, err := ctx.FileExists("package.json")
	if err != nil {
		return nil, err
	}
	if !pkgJSONExists {
		return gcp.OptOutFileNotFound("package.json"), nil
	}

	return gcp.OptIn("found yarn.lock and package.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := installYarn(ctx); err != nil {
		return fmt.Errorf("installing Yarn: %w", err)
	}

	if yarn2, err := nodejs.IsYarn2(ctx.ApplicationRoot()); err != nil {
		return err
	} else if yarn2 {
		if err := yarn2InstallModules(ctx); err != nil {
			return err
		}
	} else {
		if err := yarn1InstallModules(ctx); err != nil {
			return err
		}
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	cmd := []string{"yarn", "run", "start"}

	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess(cmd)
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	if err := devmode.AddFileWatcherProcess(ctx, devmode.Config{
		RunCmd: cmd,
		Ext:    devmode.NodeWatchedExtensions,
	}); err != nil {
		return fmt.Errorf("adding devmode file watcher: %w", err)
	}
	devmode.AddSyncMetadata(ctx, devmode.NodeSyncRules)

	return nil
}

func yarn1InstallModules(ctx *gcp.Context) error {
	freezeLockfile, err := nodejs.UseFrozenLockfile(ctx)
	if err != nil {
		return err
	}

	ml, err := ctx.Layer("yarn_modules", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	cached, err := nodejs.CheckCache(ctx, ml, cache.WithFiles("package.json", nodejs.YarnLock))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		// The yarn.lock hasn't been updated since we last built so the cached node_modules should be
		// up-to-date.
		ctx.CacheHit(cacheTag)
	} else {
		// The dependencies listed in the yarn.lock file have changed. Clear the layer cache and update
		// it after we run yarn install
		ctx.CacheMiss(cacheTag)
		if err := ctx.ClearLayer(ml); err != nil {
			return fmt.Errorf("clearing layer %q: %w", ml.Name, err)
		}
	}

	// Use Yarn's --modules-folder flag to install directly into the layer and then symlink them into
	// the app dir.
	layerModules := filepath.Join(ml.Path, "node_modules")
	appModules := filepath.Join(ctx.ApplicationRoot(), "node_modules")
	if err := ctx.MkdirAll(layerModules, 0755); err != nil {
		return err
	}
	if err := ctx.RemoveAll(appModules); err != nil {
		return err
	}
	if err := ctx.Symlink(layerModules, appModules); err != nil {
		return err
	}
	locationFlag := fmt.Sprintf("--modules-folder=%s", layerModules)

	runtimeconfigJSONExists, err := ctx.FileExists(".runtimeconfig.json")
	if err != nil {
		return err
	}
	// This is a hack to fix a bug in an old version of Firebase that loaded a config using a path
	// relative to node_modules: https://github.com/firebase/firebase-functions/issues/630.
	if runtimeconfigJSONExists {
		layerConfig := filepath.Join(ml.Path, ".runtimeconfig.json")
		if err := ctx.RemoveAll(layerConfig); err != nil {
			return err
		}
		if err := ctx.Symlink(filepath.Join(ctx.ApplicationRoot(), ".runtimeconfig.json"), layerConfig); err != nil {
			return err
		}
	}

	// Always run yarn install to execute customer's lifecycle hooks.
	cmd := []string{"yarn", "install", "--non-interactive", "--prefer-offline", locationFlag}

	// HACK: For backwards compatibility on App Engine Node.js 10 and older, skip using `--frozen-lockfile`.
	if freezeLockfile {
		cmd = append(cmd, "--frozen-lockfile")
	}
	gcpBuild, err := nodejs.HasGCPBuild(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if gcpBuild {
		// Setting --production=false causes the devDependencies to be installed regardless of the
		// NODE_ENV value. The allows the customer's lifecycle hooks to access to them. We purge the
		// devDependencies from the final app.
		cmd = append(cmd, "--production=false")
	}

	// Add the layer's node_modules/.bin to the path so it is available in postinstall scripts.
	nodeBin := filepath.Join(layerModules, ".bin")
	ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv(fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), nodeBin)))

	if gcpBuild {
		ctx.Exec([]string{"yarn", "run", "gcp-build"}, gcp.WithUserAttribution)

		// If there was a gcp-build script we installed all the devDependencies above. We should try to
		// prune them from the final app image.
		nodeEnv := nodejs.NodeEnv()
		if nodejs.NodeEnv() != nodejs.EnvProduction {
			ctx.Logf("Retaining devDependencies because NODE_ENV=%q", nodeEnv)
		} else {
			// For Yarn1, setting `--production=true` causes all `devDependencies` to be deleted.
			ctx.Logf("Pruning devDependencies")
			cmd := []string{"yarn", "install", "--ignore-scripts", "--prefer-offline", "--production=true", locationFlag}
			if freezeLockfile {
				cmd = append(cmd, "--frozen-lockfile")
			}
			ctx.Exec(cmd, gcp.WithUserAttribution)
		}
	}

	return nil
}

func yarn2InstallModules(ctx *gcp.Context) error {
	cmd := []string{"yarn", "install", "--immutable"}

	yarnCacheExists, err := ctx.FileExists(ctx.ApplicationRoot(), ".yarn", "cache")
	if err != nil {
		return err
	}
	// In Plug'n'Play mode (https://yarnpkg.com/features/pnp) all dependencies must be included in
	// the Yarn cache. The --immutable-cache option will abort the install with an error if anything
	// is missing or out of date.
	if yarnCacheExists {
		cmd = append(cmd, "--immutable-cache")
	}
	ctx.Exec(cmd, gcp.WithUserAttribution)

	// Run the gcp-build script if it exists.
	if gcpBuild, err := nodejs.HasGCPBuild(ctx.ApplicationRoot()); err != nil {
		return err
	} else if gcpBuild {
		ctx.Exec([]string{"yarn", "run", "gcp-build"}, gcp.WithUserAttribution)
	}

	// If there are no devDependencies, there is nothing to prune. We are done.
	if devDeps, err := nodejs.HasDevDependencies(ctx.ApplicationRoot()); err != nil || !devDeps {
		return err
	}

	nodeEnv := nodejs.NodeEnv()
	switch {
	case nodeEnv != nodejs.EnvProduction:
		ctx.Logf("Retaining devDependencies because NODE_ENV=%q", nodeEnv)
	case !nodejs.HasYarnWorkspacePlugin(ctx):
		ctx.Warnf("Keeping devDependencies because the Yarn workspace-tools plugin is not installed. You can add it to your project by running 'yarn plugin import workspace-tools'")
	default:
		// For Yarn2, dependency pruning is via the workspaces plugin.
		ctx.Logf("Pruning devDependencies")
		ctx.Exec([]string{"yarn", "workspaces", "focus", "--all", "--production"}, gcp.WithUserAttribution)
	}

	return nil
}

func installYarn(ctx *gcp.Context) error {
	version, err := nodejs.DetectYarnVersion(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	yrl, err := ctx.Layer(yarnLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", yarnLayer, err)
	}
	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(yrl, versionKey)
	if version == metaVersion {
		ctx.CacheHit(yarnLayer)
		ctx.Logf("Yarn cache hit, skipping installation.")
	} else {
		ctx.CacheMiss(yarnLayer)
		if err := ctx.ClearLayer(yrl); err != nil {
			return fmt.Errorf("clearing layer %q: %w", yrl.Name, err)
		}
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
	if err := ctx.Setenv("PATH", filepath.Join(yrl.Path, "bin")+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     yarnLayer,
		Metadata: map[string]interface{}{"version": version},
		Launch:   true,
		Build:    true,
	})
	return nil
}
