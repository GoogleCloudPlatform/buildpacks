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
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	cacheTag                = "prod dependencies"
	googleNodeRunScriptsEnv = "GOOGLE_NODE_RUN_SCRIPTS"
	nodejsNPMBuildEnv       = "GOOGLE_EXPERIMENTAL_NODEJS_NPM_BUILD_ENABLED"
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

	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if err := upgradeNPM(ctx, pjs); err != nil {
		return err
	}

	lockfile, err := nodejs.EnsureLockfile(ctx)
	if err != nil {
		return err
	}

	buildCmds, isCustomBuild := determineBuildCommands(pjs)
	// Respect the user's NODE_ENV value if it's set
	nodeEnv, nodeEnvPresent := os.LookupEnv(nodejs.EnvNodeEnv)
	if !nodeEnvPresent {
		if len(buildCmds) > 0 {
			// Assume that dev dependencies are required to run build scripts to
			// support the most use cases possible.
			nodeEnv = nodejs.EnvDevelopment
		} else {
			nodeEnv = nodejs.EnvProduction
		}
	}
	cached, err := nodejs.CheckOrClearCache(ctx, ml, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", lockfile))
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
		if _, err := ctx.Exec([]string{"npm", "install", "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution); err != nil {
			return err
		}
	} else {
		ctx.Logf("Installing application dependencies.")
		installCmd, err := nodejs.NPMInstallCommand(ctx)
		if err != nil {
			return err
		}

		if _, err := ctx.Exec([]string{"npm", installCmd, "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution); err != nil {
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

	if len(buildCmds) > 0 {
		// If there are multiple build scripts to run, run them one-by-one so the logs are
		// easier to understand.
		for _, cmd := range buildCmds {
			split := strings.Split(cmd, " ")
			if _, err := ctx.Exec(split, gcp.WithUserAttribution); err != nil {
				if !isCustomBuild {
					ctx.Logf("NOTE: Running the default build script can be skipped by passing the empty environment variable `%s=` to the build.", googleNodeRunScriptsEnv)
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
	el.SharedEnvironment.Default("NODE_ENV", nodeEnv)

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

// determineBuildCommands returns a list of "npm run" commands to be executed during the build
// and a bool representing whether this is a "custom build" (user-specified build scripts)
// or a system build step (default build behavior).
//
// Users can specify npm scripts to run in three ways, with the following order of precedence:
// 1. GOOGLE_NODE_RUN_SCRIPTS env var
// 2. "gcp-build" script in package.json
// 3. "build" script in package.json
func determineBuildCommands(pjs *nodejs.PackageJSON) ([]string, bool) {
	cmds := []string{}
	envScript, envScriptPresent := os.LookupEnv(googleNodeRunScriptsEnv)
	if envScriptPresent {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmGoogleNodeRunScriptsUsageCounterID).Increment(1)
		// Setting `GOOGLE_NODE_RUN_SCRIPTS=` preserves legacy behavior where "npm run build" was NOT
		// run, even though "build" was provided.
		if strings.TrimSpace(envScript) == "" {
			return []string{}, true
		}

		scripts := strings.Split(envScript, ",")
		for _, s := range scripts {
			cmds = append(cmds, fmt.Sprintf("npm run %s", strings.TrimSpace(s)))
		}

		return cmds, true
	}

	if nodejs.HasGCPBuild(pjs) {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmGcpBuildUsageCounterID).Increment(1)
		return []string{"npm run gcp-build"}, true
	}

	if pjs != nil && pjs.Scripts.Build != "" {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmBuildUsageCounterID).Increment(1)

		// If using the OSS builder, run "npm run build" by default.
		if os.Getenv(env.XGoogleTargetPlatform) == "" {
			return []string{"npm run build"}, false
		}

		// Env var guards an experimental feature to run "npm run build" by default.
		shouldBuild, err := strconv.ParseBool(os.Getenv(nodejsNPMBuildEnv))
		// If there was an error reading the env var, don't run the script.
		if err != nil {
			return []string{}, false
		}

		if shouldBuild {
			return []string{"npm run build"}, false
		}
	}

	return []string{}, false
}

func shouldPrune(ctx *gcp.Context, pjs *nodejs.PackageJSON) (bool, error) {
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
