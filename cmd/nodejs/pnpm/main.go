// Copyright 2023 Google LLC
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

// Implements nodejs/pnpm buildpack.
// The pnpm buildpack installs dependencies using pnpm and installs pnpm itself if not present.
package main

import (
	"os"
	"path/filepath"

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
	cmd := []string{"pnpm", "install"}
	if _, err := ctx.Exec(cmd, gcp.WithUserAttribution, gcp.WithEnv("CI=true")); err != nil {
		return gcp.UserErrorf("installing pnpm dependencies: %w", err)
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
