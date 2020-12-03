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

// Implements nodejs/npm_gcp_build buildpack.
// The npm_gcp_build buildpack runs the 'gcp-build' script in package.json using npm.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	cacheTag string = "dev dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.FileExists("package.json") {
		return gcp.OptOutFileNotFound("package.json"), nil
	}

	p, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %w", err)
	}
	if p.Scripts.GCPBuild == "" {
		return gcp.OptOut("gcp-build script not found in package.json"), nil
	}

	return gcp.OptIn("found package.json with a gcp-build script"), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("npm", gcp.CacheLayer)
	nm := filepath.Join(l.Path, "node_modules")
	ctx.RemoveAll("node_modules")

	nodeEnv := nodejs.EnvDevelopment
	lockfile := nodejs.EnsureLockfile(ctx, nodeEnv)

	cached, err := nodejs.CheckCache(ctx, l, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", lockfile))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)
		// Restore cached node_modules.
		ctx.Exec([]string{"cp", "--archive", nm, "node_modules"}, gcp.WithUserTimingAttribution)
	} else {
		ctx.CacheMiss(cacheTag)
		// Clear cached node_modules to ensure we don't end up with outdated dependencies.
		ctx.ClearLayer(l)
		ctx.Exec([]string{"npm", nodejs.NPMInstallCommand(ctx), "--quiet"}, gcp.WithEnv("NODE_ENV="+nodeEnv), gcp.WithUserAttribution)
		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm}, gcp.WithUserTimingAttribution)
	}

	ctx.Exec([]string{"npm", "run", "gcp-build"}, gcp.WithUserAttribution)
	ctx.RemoveAll("node_modules")
	return nil
}
