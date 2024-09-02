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

// Implements nodejs/runtime buildpack.
// The runtime buildpack installs the Node.js runtime.
package main

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	nodeLayer           = "node"
	runtimeVersionLabel = "runtime_version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	result := runtime.CheckOverride("nodejs")
	isRailsApp, _ := ruby.NeedsRailsAssetPrecompile(ctx)

	// certain Ruby on Rails apps (< 7.x) require Node.js for asset precompilation
	if !isRailsApp && result != nil {
		return result, nil
	}

	pkgJSONExists, err := ctx.FileExists("package.json")
	if err != nil {
		return nil, err
	}
	if pkgJSONExists {
		return gcp.OptInFileFound("package.json"), nil
	}
	jsFiles, err := ctx.Glob("*.js")
	if err != nil {
		return nil, fmt.Errorf("finding js files: %w", err)
	}
	if len(jsFiles) > 0 {
		return gcp.OptIn("found .js files"), nil
	}

	return gcp.OptOut("neither package.json nor any .js files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	version, err := nodejs.RequestedNodejsVersion(ctx, pjs)
	if err != nil {
		return err
	}

	if _, ok := os.LookupEnv(env.FirebaseOutputDir); ok {
		osName := runtime.OSForStack(ctx)
		latestAvailableVersion, err := runtime.ResolveVersion(ctx, runtime.Nodejs, version, osName)
		if err != nil {
			return fmt.Errorf("resolving version %s: %w", version, err)
		}
		majorVersion, err := nodejs.MajorVersion(latestAvailableVersion)
		if err != nil {
			return fmt.Errorf("getting major version for %s: %w", latestAvailableVersion, err)
		}
		ctx.AddLabel(runtimeVersionLabel, string(runtime.Nodejs)+majorVersion)
	}

	nrl, err := ctx.Layer(nodeLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", nodeLayer, err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Nodejs, version, nrl)
	return err
}
