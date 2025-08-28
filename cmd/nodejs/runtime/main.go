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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	nodeLayer           = "node"
	heapsizeLayer       = "heapsize"
	runtimeVersionLabel = "runtime_version"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn detects if package.json or .js files are present.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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

// BuildFn installs the Node.js runtime.
func BuildFn(ctx *gcp.Context) error {
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
	if _, err = runtime.InstallTarballIfNotCached(ctx, runtime.Nodejs, version, nrl); err != nil {
		return fmt.Errorf("installing nodejs: %w", err)
	}
	setHeapSize, err := env.IsPresentAndTrue("X_GOOGLE_SET_NODE_HEAP_SIZE")
	if err != nil {
		return fmt.Errorf("checking X_GOOGLE_SET_NODE_HEAP_SIZE: %w", err)
	}
	if setHeapSize {
		if err = installHeapsizeScript(ctx); err != nil {
			return fmt.Errorf("installing heapsize script: %w", err)
		}
	}
	return err
}

// installHeapsizeScript copies the exec/heapsize.sh script into the layer's exec.d directory.
func installHeapsizeScript(ctx *gcp.Context) error {
	l, err := ctx.Layer(heapsizeLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", heapsizeLayer, err)
	}
	ctx.Logf("Installing the heapsize.sh exec.d script.")
	scriptPath := filepath.Join(ctx.BuildpackRoot(), "exec", "heapsize.sh")
	destPath := filepath.Join(l.Exec.Path, "heapsize.sh")
	data, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", scriptPath, err)
	}
	ctx.MkdirAll(l.Exec.Path, 0755)
	if err := ioutil.WriteFile(destPath, data, 0777); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	return nil
}
