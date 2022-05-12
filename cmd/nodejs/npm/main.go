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

// Implements nodejs/npm buildpack.
// The npm buildpack installs dependencies using npm.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	cacheTag = "prod dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	pkgJSONExists, err := ctx.FileExists("package.json")
	if err != nil {
		return nil, err
	}
	if !pkgJSONExists {
		return gcp.OptOutFileNotFound("package.json"), nil
	}
	return gcp.OptInFileFound("package.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	ml, err := ctx.Layer("npm_modules", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	nm := filepath.Join(ml.Path, "node_modules")
	if err := ctx.RemoveAll("node_modules"); err != nil {
		return err
	}
	if err := ar.GenerateNPMConfig(ctx); err != nil {
		return fmt.Errorf("generating Artifact Registry credentials: %w", err)
	}

	if err := upgradeNPM(ctx); err != nil {
		return err
	}

	lockfile, err := nodejs.EnsureLockfile(ctx)
	if err != nil {
		return err
	}

	nodeEnv := nodejs.NodeEnv()
	gcpBuild, err := nodejs.HasGCPBuild(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if gcpBuild {
		nodeEnv = nodejs.EnvDevelopment
	}
	cached, err := nodejs.CheckCache(ctx, ml, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", lockfile))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)
		// Restore cached node_modules.
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"}, gcp.WithUserTimingAttribution)

		// Always run npm install to run preinstall/postinstall scripts.
		// Otherwise it should be a no-op because the lockfile is unchanged.
		ctx.Exec([]string{"npm", "install", "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution)
	} else {
		installCmd, err := nodejs.NPMInstallCommand(ctx)
		if err != nil {
			return err
		}
		ctx.CacheMiss(cacheTag)
		// Clear cached node_modules to ensure we don't end up with outdated dependencies after copying.
		if err := ctx.ClearLayer(ml); err != nil {
			return fmt.Errorf("clearing layer %q: %w", ml.Name, err)
		}

		ctx.Exec([]string{"npm", installCmd, "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution)

		// Ensure node_modules exists even if no dependencies were installed.
		if err := ctx.MkdirAll("node_modules", 0755); err != nil {
			return err
		}
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution)
	}

	if gcpBuild {
		ctx.Exec([]string{"npm", "run", "gcp-build"}, gcp.WithUserAttribution)

		shouldPrune, err := shouldPrune(ctx)
		if err != nil {
			return err
		}
		if shouldPrune {
			// npm prune deletes devDependencies from node_modules
			ctx.Exec([]string{"npm", "prune"}, gcp.WithUserAttribution)
		}
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	cmd := []string{"npm", "start"}

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

func shouldPrune(ctx *gcp.Context) (bool, error) {
	// if there are no devDependencies, there is no need to prune.
	if devDeps, err := nodejs.HasDevDependencies(ctx.ApplicationRoot()); err != nil || !devDeps {
		return false, err
	}
	if nodeEnv := nodejs.NodeEnv(); nodeEnv != nodejs.EnvProduction {
		ctx.Logf("Retaining devDependencies because $NODE_ENV=%q.", nodeEnv)
		return false, nil
	}
	canPrune, err := nodejs.SupportsNPMPrune(ctx)
	if err == nil && !canPrune {
		ctx.Warnf("Retaining devDependencies because the version of NPM you are using does not support 'npm prune'.")
	}
	return canPrune, err
}

func upgradeNPM(ctx *gcp.Context) error {
	npmVersion, err := nodejs.RequestedNPMVersion(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if npmVersion == "" {
		// if an NPM version was not requested, use whatever was bundled with Node.js.
		return nil
	}
	npmLayer, err := ctx.Layer("npm", gcp.BuildLayer, gcp.LaunchLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	metaVersion := ctx.GetMetadata(npmLayer, "version")
	if metaVersion == npmVersion {
		ctx.Logf("npm@%s cache hit, skipping installation.", npmVersion)
		return nil
	}
	ctx.ClearLayer(npmLayer)
	prefix := fmt.Sprintf("--prefix=%s", npmLayer.Path)
	pkg := fmt.Sprintf("npm@%s", npmVersion)
	ctx.Exec([]string{"npm", "install", "-g", prefix, pkg}, gcp.WithUserAttribution)
	// Set the path here to ensure the version we just installed takes precedence over the npm bundled
	// with the Node.js engine.
	if err := ctx.Setenv("PATH", filepath.Join(npmLayer.Path, "bin")+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}
