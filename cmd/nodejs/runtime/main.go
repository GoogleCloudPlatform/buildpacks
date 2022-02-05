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
	"path"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	nodeLayer     = "node"
	nodeURL       = "https://nodejs.org/dist/v%[1]s/node-v%[1]s-linux-x64.tar.xz"
	versionKey    = "version"
	npmVersionKey = "npm-version"
	// TODO(b/171347385): Remove after resolving incompatibilities in Node.js 15.
	defaultRange = "14.x.x"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "nodejs"); result != nil {
		return result, nil
	}

	if ctx.FileExists("package.json") {
		return gcp.OptInFileFound("package.json"), nil
	}
	if len(ctx.Glob("*.js")) > 0 {
		return gcp.OptIn("found .js files"), nil
	}

	return gcp.OptOut("neither package.json nor any .js files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	pkgJSON, err := getPkgJSONIfPresent(ctx)
	if err != nil {
		return err
	}
	version, err := runtimeVersion(ctx, pkgJSON)
	if err != nil {
		return err
	}
	npmVersion := getNPMVersion(pkgJSON)
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     nodeLayer,
		Metadata: map[string]interface{}{"version": version},
		Launch:   true,
		Build:    true,
	})

	// Check the metadata in the cache layer to determine if we need to proceed.
	nrl := ctx.Layer(nodeLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if isCached(ctx, nrl, version, npmVersion) {
		ctx.CacheHit(nodeLayer)
		ctx.Logf("Runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(nodeLayer)
	ctx.ClearLayer(nrl)

	archiveURL := fmt.Sprintf(nodeURL, version)
	code, err := ctx.HTTPStatus(archiveURL)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	// Download and install Node.js in layer.
	ctx.Logf("Installing Node.js v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xJ --directory %s --strip-components=1", archiveURL, nrl.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	if npmVersion != "" {
		ctx.Logf("Installing NPM version '%s'", npmVersion)
		installNPM(ctx, nrl, npmVersion)
		ctx.SetMetadata(nrl, npmVersionKey, npmVersion)
	}
	ctx.SetMetadata(nrl, versionKey, version)
	return nil
}

// runtimeVersion returns the version of the runtime to install.
// The version is read from env var if set or determined based on the `engines` field in package.json.
func runtimeVersion(ctx *gcp.Context, pkgJSON *nodejs.PackageJSON) (string, error) {
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	// The default empty range returns the latest version.
	var versionRange string
	if pkgJSON != nil {
		versionRange = pkgJSON.Engines.Node
	}
	if versionRange == "" {
		versionRange = defaultRange
	}
	// Use package.json and semver.io to determine best-fit Node.js version.
	ctx.Logf("Resolving Node.js version based on semver %q", versionRange)
	result := ctx.Exec([]string{"curl", "--fail", "--show-error", "--silent", "--location", "--get", "--data-urlencode", fmt.Sprintf("range=%s", versionRange), "http://semver.io/node/resolve"}, gcp.WithUserAttribution)
	version := result.Stdout
	ctx.Logf("Using resolved runtime version from package.json: %s", version)
	return version, nil
}

// getNPMVersion returns the NPM version from the 'engines' field, if not present or the empty string then an empty string is returned.
func getNPMVersion(pkgJSON *nodejs.PackageJSON) string {
	if pkgJSON == nil {
		return ""
	}
	return pkgJSON.Engines.NPM
}

// isCached returns true if the requested version of node and NPM match what is isntalled into the 'nrl' layer
func isCached(ctx *gcp.Context, nrl *libcnb.Layer, nodeVersion, npmVersion string) bool {
	metaVersion := ctx.GetMetadata(nrl, versionKey)
	if metaVersion != nodeVersion {
		return false
	}
	metaNPMVersion := ctx.GetMetadata(nrl, npmVersionKey)
	return metaNPMVersion == npmVersion
}

// installNPM installs NPM into the system, respecting the version specified
func installNPM(ctx *gcp.Context, nrl *libcnb.Layer, version string) {
	nodeBin := path.Join(nrl.Path, "bin")
	npmBinaryPath := path.Join(nodeBin, "npm")
	ctx.Exec([]string{npmBinaryPath, "install", fmt.Sprintf("npm@%v", version), "-g"},
		gcp.WithEnv(fmt.Sprintf("PATH=${PATH}:%v", nodeBin)),
		gcp.WithUserAttribution)
}

func getPkgJSONIfPresent(ctx *gcp.Context) (*nodejs.PackageJSON, error) {
	if !ctx.FileExists("package.json") {
		return nil, nil
	}
	return nodejs.ReadPackageJSON(ctx.ApplicationRoot())
}
