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
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	cacheTag = "prod dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("package.json") {
		ctx.OptOut("package.json not found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	ml := ctx.Layer("npm")
	nm := filepath.Join(ml.Root, "node_modules")
	ctx.RemoveAll("node_modules")
	nodejs.EnsurePackageLock(ctx)

	nodeEnv := nodejs.NodeEnv()
	cached, meta, err := nodejs.CheckCache(ctx, ml, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", nodejs.PackageLock))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)
		// Restore cached node_modules.
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"})

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
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm})
	}

	ctx.WriteMetadata(ml, &meta, layers.Build, layers.Cache)

	el := ctx.Layer("env")
	ctx.PrependPathSharedEnv(el, "PATH", filepath.Join(ctx.ApplicationRoot(), "node_modules", ".bin"))
	ctx.DefaultSharedEnv(el, "NODE_ENV", nodeEnv)
	ctx.WriteMetadata(el, nil, layers.Launch, layers.Build)

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
