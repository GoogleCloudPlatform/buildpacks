// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	 http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implements nodejs/bun buildpack.
// The bun buildpack installs dependencies using bun and installs bun itself if not present.
package lib

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	cacheTag = "prod dependencies"
	bunLayer = "bun_engine"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	pkgJSONExists, err := ctx.FileExists("package.json")
	if err != nil {
		return nil, err
	}
	if !pkgJSONExists {
		return gcp.OptOutFileNotFound("package.json"), nil
	}

	bunLockExists, err := ctx.FileExists(nodejs.BunLock)
	if err != nil {
		return nil, err
	}
	bunLockBExists, err := ctx.FileExists(nodejs.BunLockB)
	if err != nil {
		return nil, err
	}
	if !bunLockExists && !bunLockBExists {
		return gcp.OptOutFileNotFound("bun.lock or bun.lockb"), nil
	}

	return gcp.OptIn("found bun lockfile and package.json"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.BunUsageCounterID).Increment(1)
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if err := installBun(ctx, pjs); err != nil {
		return gcp.InternalErrorf("installing bun: %w", err)
	}

	if err := bunInstallModules(ctx, pjs); err != nil {
		return err
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return gcp.InternalErrorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	ctx.AddWebProcess([]string{"bun", "run", "start"})
	return nil
}

func bunInstallModules(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	pjs, err := nodejs.OverrideAppHostingBuildScript(ctx, nodejs.ApphostingPreprocessedPathForPack)
	if err != nil {
		return err
	}
	buildCmds, _ := nodejs.DetermineBuildCommands(pjs, "bun")
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
	cmd := []string{"bun", "install"}
	if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("CI=true"), gcp.WithEnv("NODE_ENV="+buildNodeEnv)); err != nil {
		return gcp.UserErrorf("installing bun dependencies: %w", err)
	}
	if len(buildCmds) > 0 {
		// If there are multiple build scripts to run, run them one-by-one so the logs are
		// easier to understand.
		for _, cmd := range buildCmds {
			split := strings.Split(cmd, " ")
			if _, err := ctx.Exec(split, gcp.WithUserAttribution); err != nil {
				if fahCmd, fahCmdPresent := os.LookupEnv(nodejs.AppHostingBuildEnv); fahCmdPresent {
					return gcp.UserErrorf("%w", faherror.FailedFrameworkBuildError(fahCmd, err))
				}
				if nodejs.HasApphostingPackageBuild(pjs) {
					return gcp.UserErrorf("%w", faherror.FailedFrameworkBuildError(pjs.Scripts[nodejs.ScriptApphostingBuild], err))
				}
				return err
			}
		}
	}
	shouldPruneDevDependencies := buildNodeEnv == nodejs.EnvDevelopment && !nodeEnvPresent && nodejs.HasDevDependencies(pjs)
	if shouldPruneDevDependencies {
		if env.IsFAH() {
			// We don't prune if the user is using App Hosting since App Hosting builds don't
			// rely on the node_modules folder at this point.
			return nil
		}
		// If we installed dependencies with NODE_ENV=development and the user didn't explicitly set
		// NODE_ENV we should prune the devDependencies from the final app image.
		// Note: Bun doesn't have a native prune command, so we reinstall with --production flag
		cmd := []string{"bun", "install", "--production"}
		if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("CI=true")); err != nil {
			return gcp.UserErrorf("pruning devDependencies: %w", err)
		}
	}
	return nil
}

func installBun(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	layer, err := ctx.Layer(bunLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return gcp.InternalErrorf("creating %v layer: %w", bunLayer, err)
	}
	return nodejs.InstallBun(ctx, layer, pjs)
}
