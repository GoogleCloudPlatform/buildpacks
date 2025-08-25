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
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
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
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NPMUsageCounterID).Increment(1)
	ml, err := ctx.Layer("npm_modules", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	nm := filepath.Join(ml.Path, "node_modules")
	if nmExists, _ := ctx.FileExists("node_modules"); nmExists {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmNodeModulesCounterID).Increment(1)

	}
	vendorNpmDeps := nodejs.IsUsingVendoredDependencies()
	if !vendorNpmDeps {
		if err := ctx.RemoveAll("node_modules"); err != nil {
			return err
		}
	}
	if err := ar.GenerateNPMConfig(ctx); err != nil {
		return fmt.Errorf("generating Artifact Registry credentials: %w", err)
	}

	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if err := upgradeNPM(ctx, pjs); err != nil {
		vendorError := ""
		if vendorNpmDeps {
			vendorError = "Vendored dependencies detected, please remove the npm version from your package.json to avoid installing npm and instead use the bundled npm"
		}
		return fmt.Errorf("%s Error: %w", vendorError, err)
	}

	lockfile, err := nodejs.EnsureLockfile(ctx)
	if err != nil {
		return err
	}
	pjs, err = nodejs.OverrideAppHostingBuildScript(ctx, nodejs.ApphostingPreprocessedPathForPack)
	if err != nil {
		return err
	}
	buildCmds, isCustomBuild := nodejs.DetermineBuildCommands(pjs, "npm")
	// Respect the user's NODE_ENV value if it's set
	buildNodeEnv, nodeEnvPresent := os.LookupEnv(nodejs.EnvNodeEnv)
	if !nodeEnvPresent {
		if len(buildCmds) > 0 {
			// Assume that dev dependencies are required to run build scripts to
			// support the most use cases possible.
			buildNodeEnv = nodejs.EnvDevelopment
		} else {
			buildNodeEnv = nodejs.EnvProduction
		}
	}

	if vendorNpmDeps {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmVendorDependenciesCounterID).Increment(1)
		if _, err := ctx.Exec([]string{"npm", "rebuild"}, gcp.WithEnv("NODE_ENV="+buildNodeEnv), gcp.WithUserAttribution); err != nil {
			return err
		}
	} else {
		cached, err := nodejs.CheckOrClearCache(ctx, ml, cache.WithStrings(buildNodeEnv), cache.WithFiles("package.json", lockfile))
		if err != nil {
			return fmt.Errorf("checking cache: %w", err)
		}
		if cached {
			// Restore cached node_modules.
			if _, err := ctx.Exec([]string{"cp", "--archive", nm, "node_modules"}, gcp.WithUserTimingAttribution); err != nil {
				return err
			}

			// Always run npm install to run preinstall/postinstall scripts.
			// Otherwise it should be a no-op because the lockfile is unchanged.
			if _, err := ctx.Exec([]string{"npm", "install", "--quiet"}, gcp.WithEnv("NODE_ENV="+buildNodeEnv), gcp.WithUserAttribution); err != nil {
				return err
			}
		} else {
			ctx.Logf("Installing application dependencies.")
			installCmd, err := nodejs.NPMInstallCommand(ctx)
			if err != nil {
				return err
			}

			if _, err := ctx.Exec([]string{"npm", installCmd, "--quiet", "--no-fund", "--no-audit"}, gcp.WithEnv("NODE_ENV="+buildNodeEnv), gcp.WithUserAttribution); err != nil {
				return err
			}
			// Ensure node_modules exists even if no dependencies were installed.
			if err := ctx.MkdirAll("node_modules", 0755); err != nil {
				return err
			}
			if _, err := ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution); err != nil {
				return err
			}
		}
	}

	if len(buildCmds) > 0 {
		// If there are multiple build scripts to run, run them one-by-one so the logs are
		// easier to understand.
		for _, cmd := range buildCmds {
			execOpts := []gcp.ExecOption{gcp.WithUserAttribution}
			if nodejs.DetectSvelteKitAutoAdapter(pjs) {
				execOpts = append(execOpts, gcp.WithEnv(nodejs.SvelteAdapterEnv))
			}
			split := strings.Split(cmd, " ")
			if _, err := ctx.Exec(split, execOpts...); err != nil {
				if !isCustomBuild {
					return fmt.Errorf(`%w
NOTE: Running the default build script can be skipped by passing the empty environment variable "%s=" to the build`, err, nodejs.GoogleNodeRunScriptsEnv)
				}
				if fahCmd, fahCmdPresent := os.LookupEnv(nodejs.AppHostingBuildEnv); fahCmdPresent {
					return gcp.UserErrorf("%w", faherror.FailedFrameworkBuildError(fahCmd, err))
				}
				if nodejs.HasApphostingPackageBuild(pjs) {
					return gcp.UserErrorf("%w", faherror.FailedFrameworkBuildError(pjs.Scripts[nodejs.ScriptApphostingBuild], err))
				}
				return err
			}
		}

		shouldPrune, err := shouldPrune(ctx, pjs)
		if err != nil {
			return err
		}
		if shouldPrune {
			// npm prune deletes devDependencies from node_modules
			if _, err := ctx.Exec([]string{"npm", "prune", "--production"}, gcp.WithUserAttribution); err != nil {
				return err
			}
		}
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	cmd, err := nodejs.DefaultStartCommand(ctx, pjs)
	if err != nil {
		return fmt.Errorf("detecting start command: %w", err)
	}

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

	return nil
}

func shouldPrune(ctx *gcp.Context, pjs *nodejs.PackageJSON) (bool, error) {
	// if we are vendoring dependencies, we do not need to prune
	if nodejs.IsUsingVendoredDependencies() {
		return false, nil
	}

	// if there are no devDependencies, there is no need to prune.
	if !nodejs.HasDevDependencies(pjs) {
		return false, nil
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

func upgradeNPM(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	npmVersion, err := nodejs.RequestedNPMVersion(pjs)
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
	if _, err := ctx.Exec([]string{"npm", "install", "-g", prefix, pkg}, gcp.WithUserAttribution); err != nil {
		return err
	}
	// Set the path here to ensure the version we just installed takes precedence over the npm bundled
	// with the Node.js engine.
	if err := ctx.Setenv("PATH", filepath.Join(npmLayer.Path, "bin")+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}
