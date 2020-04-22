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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	sdkLayer     = "sdk"
	runtimeLayer = "runtime"
	sdkURL       = "https://dotnetcli.azureedge.net/dotnet/Sdk/%[1]s/dotnet-sdk-%[1]s-linux-x64.tar.gz"
	versionURL   = "https://dotnetcli.azureedge.net/dotnet/Sdk/LTS/latest.version"
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

	if len(dotnet.ProjectFiles(ctx, ".")) == 0 && !ctx.HasAtLeastOne("*.dll") {
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
	var sdkMeta metadata
	sdkl := ctx.Layer(sdkLayer)
	ctx.ReadMetadata(sdkl, &sdkMeta)

	var rtMeta metadata
	rtl := ctx.Layer(runtimeLayer)
	ctx.ReadMetadata(rtl, &rtMeta)

	// Each SDK is associated with one Core version, but the reverse is not true.
	// We use the SDK version as the "runtime" version.
	if version == sdkMeta.Version && version == rtMeta.Version {
		ctx.CacheHit(sdkLayer)
		ctx.CacheHit(runtimeLayer)
		ctx.Logf(".NET cache hit, skipping installation.")
		return nil
	}

	ctx.CacheMiss(sdkLayer)
	ctx.ClearLayer(sdkl)

	ctx.CacheMiss(runtimeLayer)
	ctx.ClearLayer(rtl)

	archiveURL := fmt.Sprintf(sdkURL, version)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	ctx.Logf("Installing .NET SDK v%s", version)
	// Ensure there's a symlink from runtime/sdk dir to the sdk layer.
	// TODO: remove the symlink in the final image.
	ctx.Exec([]string{"ln", "--symbolic", "--force", sdkl.Root, filepath.Join(rtl.Root, "sdk")})

	// With --keep-directory-symlink, the SDK will be unpacked into /runtime/sdk,
	// which is symlinked to the SDK layer. This is needed because the dotnet CLI
	// needs an sdk directory in the same directory as the dotnet executable.
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --keep-directory-symlink --strip-components=1", archiveURL, rtl.Root)
	ctx.Exec([]string{"bash", "-c", command})

	// Keep the SDK layer for launch in devmode because we use `dotnet watch`.
	sdkMeta.Version = version
	if devmode.Enabled(ctx) {
		ctx.WriteMetadata(sdkl, sdkMeta, layers.Launch, layers.Build, layers.Cache)
	} else {
		ctx.WriteMetadata(sdkl, sdkMeta, layers.Build, layers.Cache)
	}

	rtMeta.Version = version
	ctx.DefaultSharedEnv(rtl, "DOTNET_ROOT", rtl.Root)
	ctx.PrependPathSharedEnv(rtl, "PATH", rtl.Root)
	ctx.DefaultLaunchEnv(rtl, "DOTNET_RUNNING_IN_CONTAINER", "true")
	ctx.WriteMetadata(rtl, rtMeta, layers.Launch, layers.Build, layers.Cache)

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    runtimeLayer,
		Version: version,
	})

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
