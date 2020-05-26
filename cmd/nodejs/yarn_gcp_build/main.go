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

// Implements nodejs/yarn_gcp_build buildpack.
// The npm_gcp_build buildpack runs the 'gcp-build' script in package.json using yarn.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	cacheTag string = "dev dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists(nodejs.YarnLock) {
		ctx.OptOut("yarn.lock not found.")
	}
	if !ctx.FileExists("package.json") {
		ctx.OptOut("package.json not found.")
	}

	p, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}
	if p.Scripts.GCPBuild == "" {
		ctx.OptOut("gcp-build script not found in package.json.")
	}

	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("yarn")
	nm := filepath.Join(l.Root, "node_modules")
	ctx.RemoveAll("node_modules")

	nodeEnv := nodejs.EnvDevelopment
	cached, meta, err := nodejs.CheckCache(ctx, l, cache.WithStrings(nodeEnv), cache.WithFiles("package.json", nodejs.YarnLock))
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
		// Clear cached node_modules to ensure we don't end up with outdated dependencies.
		ctx.ClearLayer(l)

		cmd := []string{"yarn", "install", "--non-interactive"}
		if lf := nodejs.LockfileFlag(ctx); lf != "" {
			cmd = append(cmd, lf)
		}
		ctx.ExecUserWithParams(gcp.ExecParams{
			Cmd: cmd,
			Env: []string{"NODE_ENV=" + nodeEnv},
		}, gcp.UserErrorKeepStderrTail)

		// Ensure node_modules exists even if no dependencies were installed.
		ctx.MkdirAll("node_modules", 0755)
		ctx.Exec([]string{"cp", "--archive", "node_modules", nm})
	}

	ctx.ExecUser([]string{"yarn", "run", "gcp-build"})
	ctx.RemoveAll("node_modules")
	ctx.WriteMetadata(l, &meta, layers.Cache)
	return nil
}
