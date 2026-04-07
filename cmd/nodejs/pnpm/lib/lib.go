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

// Implements nodejs/pnpm buildpack.
// The pnpm buildpack installs dependencies using pnpm and installs pnpm itself if not present.
package lib

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	cacheTag  = "prod dependencies"
	pnpmLayer = "pnpm_engine"
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

	pnpmLockExists, err := ctx.FileExists(nodejs.PNPMLock)
	if err != nil {
		return nil, err
	}
	if pnpmLockExists {
		return gcp.OptIn("found pnpm-lock.yaml and package.json"), nil
	}

	if nodejs.IsPackageManagerConfigured("pnpm") {
		return gcp.OptIn("package.json found and GOOGLE_PACKAGE_MANAGER=pnpm"), nil
	}

	return gcp.OptOut("pnpm-lock.yaml not found and GOOGLE_PACKAGE_MANAGER is not set to pnpm"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, buildermetadata.MetadataValue("pnpm"))
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

	// Check for React2Shell vulnerability in the lockfile.
	nodeDeps, err := nodejs.ReadNodeDependencies(ctx, ctx.ApplicationRoot())
	if err != nil {
		ctx.Warnf("Failed to read node dependencies: %v", err)
	} else {
		if err := nodejs.CheckVulnerabilities(ctx, nodeDeps); err != nil {
			return err
		}
	}

	// Configure the entrypoint for production.
	entrypoint, err := nodejs.Entrypoint(ctx, "pnpm")
	if err != nil {
		return err
	}

	devSync, err := env.IsDevSync()
	if err != nil {
		ctx.Warnf("Unable to determine dev sync status: %v", err)
	} else if devSync {
		entrypoint, err = nodejs.DevSyncEntrypoint(ctx, pjs, "pnpm")
		if err != nil {
			return gcp.InternalErrorf("getting dev sync entrypoint: %w", err)
		}
		ctx.AddWebProcess(entrypoint)
		return nil
	}

	ctx.AddWebProcess(entrypoint)
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
	if nodejs.ShouldPrunePnpmBun(ctx, pjs, buildNodeEnv, nodeEnvPresent) {
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
