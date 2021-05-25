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
	if !ctx.FileExists("package.json") {
		return gcp.OptOutFileNotFound("package.json"), nil
	}
	return gcp.OptInFileFound("package.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	ml := ctx.Layer("npm", gcp.BuildLayer, gcp.CacheLayer)
	nm := filepath.Join(ml.Path, "node_modules")
	ctx.RemoveAll("node_modules")

	lockfile := nodejs.EnsureLockfile(ctx)

	nodeEnv := nodejs.NodeEnv()
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
		ctx.CacheMiss(cacheTag)
		// Clear cached node_modules to ensure we don't end up with outdated dependencies after copying.
		ctx.ClearLayer(ml)

		ctx.Exec([]string{"npm", nodejs.NPMInstallCommand(ctx), "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution)

		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution)
	}

	el := ctx.Layer("env", gcp.BuildLayer, gcp.LaunchLayer)
	el.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	el.SharedEnvironment.Default("NODE_ENV", nodeEnv)

	// Configure the entrypoint for production.
	cmd := []string{"npm", "start"}

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
