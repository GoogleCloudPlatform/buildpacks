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
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	nodeLayer = "node"
	nodeURL   = "https://nodejs.org/dist/v%[1]s/node-v%[1]s-linux-x64.tar.xz"
)

// metadata represents metadata stored for a runtime layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	runtime.CheckOverride(ctx, "nodejs")

	if ctx.FileExists("package.json") {
		return nil
	}
	if len(ctx.Glob("*.js")) > 0 {
		return nil
	}

	ctx.OptOut("package.json not found and no *.js files found")
	return nil // OptOut() above exits early.
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta metadata
	nrl := ctx.Layer(nodeLayer)
	ctx.ReadMetadata(nrl, &meta)
	if version == meta.Version {
		ctx.CacheHit(nodeLayer)
		ctx.Logf("Runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(nodeLayer)
	ctx.ClearLayer(nrl)

	archiveURL := fmt.Sprintf(nodeURL, version)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	// Download and install Node.js in layer.
	ctx.Logf("Installing Node.js v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location %s | tar xJ --directory=%s --strip-components=1", archiveURL, nrl.Root)
	ctx.Exec([]string{"bash", "-c", command})

	meta.Version = version
	ctx.WriteMetadata(nrl, meta, layers.Build, layers.Cache, layers.Launch)

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    nodeLayer,
		Version: version,
	})
	return nil
}

// runtimeVersion returns the version of the runtime to install.
// The version is read from env var if set or determined based on the `engines` field in package.json.
func runtimeVersion(ctx *gcp.Context) (string, error) {
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	// The default empty range returns the latest version.
	var versionRange string
	if ctx.FileExists("package.json") {
		pjs, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
		if err != nil {
			return "", fmt.Errorf("reading package.json: %w", err)
		}
		versionRange = pjs.Engines.Node
	}
	// Use package.json and semver.io to determine best-fit Node.js version.
	ctx.Logf("Resolving Node.js version based on semver %q", versionRange)
	result := ctx.Exec([]string{"curl", "--silent", "--get", "--data-urlencode", fmt.Sprintf("range=%s", versionRange), "http://semver.io/node/resolve"})
	version := result.Stdout
	ctx.Logf("Using resolved runtime version from package.json: %s", version)
	return version, nil
}
