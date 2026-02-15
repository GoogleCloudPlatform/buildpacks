// Copyright 2026 Google LLC
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

// Package lib implements nodejs/bun buildpack.
// The bun buildpack installs dependencies using bun package manager.
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
	bunLayer = "bun"
)

// DetectFn detects if package.json is present.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsBetaSupported() {
		return gcp.OptOut("Bun package manager is not supported in the current release track."), nil
	}
	pkgJSONExists, err := ctx.FileExists("package.json")
	if err != nil {
		return nil, err
	}
	if !pkgJSONExists {
		return gcp.OptOutFileNotFound("package.json"), nil
	}
	bunLockbExists, err := ctx.FileExists(nodejs.BunLockb)
	if err != nil {
		return nil, err
	}
	if bunLockbExists {
		return gcp.OptInFileFound(nodejs.BunLockb), nil
	}
	bunLockExists, err := ctx.FileExists(nodejs.BunLock)
	if err != nil {
		return nil, err
	}
	if bunLockExists {
		return gcp.OptInFileFound(nodejs.BunLock), nil
	}
	if os.Getenv(env.PackageManager) == "bun" {
		return gcp.OptIn("package.json found and GOOGLE_PACKAGE_MANAGER=bun"), nil
	}
	return gcp.OptOut("bun.lockb or bun.lock not found"), nil
}

// BuildFn installs dependencies using bun package manager.
func BuildFn(ctx *gcp.Context) error {
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, buildermetadata.MetadataValue("bun"))
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	if err := installBun(ctx, pjs); err != nil {
		return gcp.InternalErrorf("installing bun: %w", err)
	}

	if err := bunInstallModules(ctx); err != nil {
		return err
	}

	el, err := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return gcp.InternalErrorf("creating layer: %w", err)
	}
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodejs.NodeEnv())

	// Configure the entrypoint for production.
	ctx.AddWebProcess([]string{"npm", "run", "start"})
	return nil
}

func bunInstallModules(ctx *gcp.Context) error {
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

	bunLockbExists, _ := ctx.FileExists("bun.lockb")
	bunLockExists, _ := ctx.FileExists("bun.lock")
	cmd := []string{"bun", "install"}
	if bunLockbExists || bunLockExists {
		ctx.Logf("Lockfile (bun.lockb or bun.lock) found. Installing dependencies with Bun using --frozen-lockfile.")
		cmd = append(cmd, "--frozen-lockfile")
	} else {
		ctx.Logf("Installing application dependencies with Bun (no lockfile found).")
	}

	if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("NODE_ENV="+buildNodeEnv)); err != nil {
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
			return nil
		}
		// If we installed dependencies with NODE_ENV=development and the user didn't explicitly set
		// NODE_ENV we should prune the DevDependencies from the final app image.
		ctx.Logf("Pruning DevDependencies.")
		cmd := []string{"bun", "install", "--frozen-lockfile", "--production"}
		if _, err := ctx.Exec(cmd, gcp.WithUserAttribution); err != nil {
			return gcp.UserErrorf("pruning devDependencies: %w", err)
		}
	}
	return nil
}

var installBun = func(ctx *gcp.Context, pjs *nodejs.PackageJSON) error {
	layer, err := ctx.Layer(bunLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return gcp.InternalErrorf("creating %v layer: %w", bunLayer, err)
	}
	return nodejs.InstallBun(ctx, layer, pjs)
}
