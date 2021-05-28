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
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	sdkLayer       = "sdk"
	runtimeLayer   = "runtime"
	sdkURL         = "https://dotnetcli.azureedge.net/dotnet/Sdk/%[1]s/dotnet-sdk-%[1]s-linux-x64.tar.gz"
	uncachedSdkURL = "https://dotnetcli.blob.core.windows.net/dotnet/Sdk/%[1]s/dotnet-sdk-%[1]s-linux-x64.tar.gz"
	versionURL     = "https://dotnetcli.blob.core.windows.net/dotnet/Sdk/LTS/latest.version"
	versionKey     = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "dotnet"); result != nil {
		return result, nil
	}

	if files := dotnet.ProjectFiles(ctx, "."); len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}
	if ctx.HasAtLeastOne("*.dll") {
		return gcp.OptIn("found .dll files"), nil
	}

	return gcp.OptOut("no project files or .dll files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}

	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     runtimeLayer,
		Metadata: map[string]interface{}{"version": version},
		Launch:   true,
		Build:    true,
	})

	sdkl := ctx.Layer(sdkLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	rtl := ctx.Layer(runtimeLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)

	// Check the metadata in the cache layer to determine if we need to proceed.
	// Each SDK is associated with one Core version, but the reverse is not true.
	// We use the SDK version as the "runtime" version.
	sdkMetaVersion := ctx.GetMetadata(sdkl, versionKey)
	rtMetaVersion := ctx.GetMetadata(rtl, versionKey)
	if version == sdkMetaVersion && version == rtMetaVersion {
		ctx.CacheHit(sdkLayer)
		ctx.CacheHit(runtimeLayer)
		ctx.Logf(".NET cache hit, skipping installation.")
		return nil
	}

	ctx.CacheMiss(sdkLayer)
	ctx.ClearLayer(sdkl)

	ctx.CacheMiss(runtimeLayer)
	ctx.ClearLayer(rtl)

	archiveURL, err := archiveURL(ctx, version)
	if err != nil {
		return err
	}

	ctx.Logf("Installing .NET SDK v%s", version)
	// Ensure there's a symlink from runtime/sdk dir to the sdk layer.
	// TODO(b/150893022): remove the symlink in the final image.
	ctx.Exec([]string{"ln", "--symbolic", "--force", sdkl.Path, filepath.Join(rtl.Path, "sdk")})

	// With --keep-directory-symlink, the SDK will be unpacked into /runtime/sdk,
	// which is symlinked to the SDK layer. This is needed because the dotnet CLI
	// needs an sdk directory in the same directory as the dotnet executable.
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --keep-directory-symlink --strip-components=1", archiveURL, rtl.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	// Keep the SDK layer for launch in devmode because we use `dotnet watch`.
	ctx.SetMetadata(sdkl, versionKey, version)
	ctx.SetMetadata(rtl, versionKey, version)
	rtl.SharedEnvironment.Default("DOTNET_ROOT", rtl.Path)
	rtl.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), rtl.Path)
	rtl.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")

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
	command := fmt.Sprintf("curl --fail --show-error --silent --location %s | tail -n 1", versionURL)
	result := ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)
	version = result.Stdout
	ctx.Logf("Using the latest LTS version of .NET Core SDK: %s", version)
	return version, nil
}

// archiveURL returns the URL to fetch the .NET SDK.
func archiveURL(ctx *gcp.Context, version string) (string, error) {
	url := fmt.Sprintf(sdkURL, version)
	if code := ctx.HTTPStatus(url); code == http.StatusOK {
		return url, nil
	}

	// Retry with the uncached URL.
	url = fmt.Sprintf(uncachedSdkURL, version)
	if code := ctx.HTTPStatus(url); code != http.StatusOK {
		return "", gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, url, code, env.RuntimeVersion)
	}

	return url, nil
}
