// Copyright 2023 Google LLC
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

// Implements nodejs/pnpm buildpack.
// The pnpm buildpack installs dependencies using pnpm and installs pnpm itself if not present.
package main

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
	cacheTag  = "prod dependencies"
	pnpmLayer = "pnpm_engine"
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

	pnpmLockExists, err := ctx.FileExists(nodejs.PNPMLock)
	if err != nil {
		return nil, err
	}
	if !pnpmLockExists {
		return gcp.OptOutFileNotFound(nodejs.PNPMLock), nil
	}

	return gcp.OptIn("found pnpm-lock.yaml and package.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PNPMUsageCounterID).Increment(1)
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if err := installPNPM(ctx, pjs); err != nil {
		return gcp.InternalErrorf("installing pnpm: %w", err)
	}

	if err := pnpmInstallModules(ctx, pjs); err != nil {
		return err
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return gcp.InternalErrorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	ctx.AddWebProcess([]string{"pnpm", "run", "start"})
	return nil
}

func pnpmInstallModules(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	pjs, err := nodejs.OverrideAppHostingBuildScript(ctx, nodejs.ApphostingPreprocessedPathForPack)
	if err != nil {
		return err
	}
	buildCmds, _ := nodejs.DetermineBuildCommands(pjs, "pnpm")
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
	cmd := []string{"pnpm", "install"}
	if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("CI=true"), gcp.WithEnv("NODE_ENV="+buildNodeEnv)); err != nil {
		return gcp.UserErrorf("installing pnpm dependencies: %w", err)
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
		cmd := []string{"pnpm", "prune", "--prod"}
		if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("CI=true")); err != nil {
			return gcp.UserErrorf("pruning devDependencies: %w", err)
		}
	}
	return nil
}

func installPNPM(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	layer, err := ctx.Layer(pnpmLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return gcp.InternalErrorf("creating %v layer: %w", pnpmLayer, err)
	}
	return nodejs.InstallPNPM(ctx, layer, pjs)
}
