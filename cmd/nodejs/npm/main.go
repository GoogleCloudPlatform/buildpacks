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

// Implements /bin/build for nodejs/npm buildpack.
package main

import (
	"fmt"
	"path"

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
	l := ctx.Layer("npm")
	nm := path.Join(l.Root, "node_modules")
	ctx.RemoveAll("node_modules")
	nodejs.EnsurePackageLock(ctx)

	cached, meta, err := nodejs.CheckCache(ctx, l, nodejs.PackageLock)
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)
		ctx.Logf("Due to cache hit, package.json scripts will not be run. To run the scripts, disable caching.")
		// Restore cached node_modules.
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"})
	} else {
		ctx.CacheMiss(cacheTag)
		ctx.ClearLayer(l)
		ctx.ExecUser([]string{"npm", nodejs.NPMInstallCommand(ctx), "--quiet", "--production"})
		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm})
	}

	ctx.PrependPathSharedEnv(l, "PATH", "node_modules/.bin")
	ctx.DefaultLaunchEnv(l, "NODE_ENV", "production")
	ctx.WriteMetadata(l, &meta, layers.Build, layers.Cache)

	// Configure the entrypoint for production.
	cmd := []string{"npm", "start"}

	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess(cmd)
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	devmode.AddFileWatcherProcess(ctx, devmode.Config{
		Cmd: cmd,
		Ext: devmode.NodeWatchedExtensions,
	})
	devmode.AddSyncMetadata(ctx, devmode.NodeSyncRules)

	return nil
}
