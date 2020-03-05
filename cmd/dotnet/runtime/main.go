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

// Binary dotnet/runtime buildpack detects .NET applications
// and install the corresponding version of .NET runtime.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	sdkLayer    = "sdk"
	dotnetLayer = "dotnet"
	sdkURL      = "https://dotnetcli.azureedge.net/dotnet/Sdk/%[1]s/dotnet-sdk-%[1]s-linux-x64.tar.gz"
	versionURL  = "https://dotnetcli.azureedge.net/dotnet/Sdk/LTS/latest.version"
)

// metadata represents metadata stored for a runtime layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	runtime.CheckOverride(ctx, "dotnet")

	if !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.*proj") && !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.dll") {
		ctx.OptOut("No project files nor .dll files found.")
	}

	return nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}
	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta metadata
	sdkl := ctx.Layer(sdkLayer)
	ctx.ReadMetadata(sdkl, &meta)
	if version == meta.Version {
		ctx.CacheHit(sdkLayer)
		ctx.Logf(".NET SDK cache hit, skipping installation.")
		return nil
	}

	ctx.CacheMiss(sdkLayer)
	ctx.ClearLayer(sdkl)

	archiveURL := fmt.Sprintf(sdkURL, version)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	ctx.Logf("Installing .NET SDK v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location %s | tar xz --directory=%s --strip-components=1", archiveURL, sdkl.Root)
	ctx.Exec([]string{"bash", "-c", command})

	meta.Version = version
	ctx.OverrideSharedEnv(sdkl, "DOTNET_ROOT", sdkl.Root)
	ctx.PrependPathSharedEnv(sdkl, "PATH", sdkl.Root)
	ctx.WriteMetadata(sdkl, meta, layers.Launch, layers.Build, layers.Cache)

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    sdkLayer,
		Version: version,
	})

	// TODO: create a dotnet runtime (no sdk) layer for launch.
	return nil
}

// globalJSON represents the contents of a global.json file.
type globalJSON struct {
	Sdk struct {
		Version string `json:"version"`
	} `json:"sdk"`
}

// runtimeVersion returns the version of the .NET Core SDK to install.
func runtimeVersion(ctx *gcp.Context) (string, error) {
	version := os.Getenv(env.RuntimeVersion)
	if version != "" {
		ctx.Logf("Using .NET Core SDK version from env: %s", version)
		return version, nil
	}

	if ctx.FileExists("global.json") {
		rawgjs, err := ioutil.ReadFile(filepath.Join(ctx.ApplicationRoot(), "global.json"))
		if err != nil {
			return "", fmt.Errorf("reading global.json: %v", err)
		}

		var gjs globalJSON
		if err := json.Unmarshal(rawgjs, &gjs); err != nil {
			return "", gcp.UserErrorf("unmarshalling global.json: %v", err)
		}

		if gjs.Sdk.Version != "" {
			ctx.Logf("Using .NET Core SDK version from global.json: %s", version)
			return gjs.Sdk.Version, nil
		}
	}

	// Use the latest LTS version.
	command := fmt.Sprintf("curl --silent %s | tail -n 1", versionURL)
	result := ctx.Exec([]string{"bash", "-c", command})
	version = result.Stdout
	ctx.Logf("Using the latest LTS version of .NET Core SDK: %s", version)
	return version, nil
}
