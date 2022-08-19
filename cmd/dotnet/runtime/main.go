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
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	runtimeLayerName         = "runtime"
	aspnetRuntimeURL         = "https://dotnetcli.azureedge.net/dotnet/aspnetcore/Runtime/%[1]s/aspnetcore-runtime-%[1]s-linux-x64.tar.gz"
	uncachedAspnetRuntimeURL = "https://dotnetcli.blob.core.windows.net/dotnet/aspnetcore/Runtime/%[1]s/aspnetcore-runtime-%[1]s-linux-x64.tar.gz"
	versionKey               = "version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("dotnet"); result != nil {
		return result, nil
	}

	files, err := dotnet.ProjectFiles(ctx, ".")
	if err != nil {
		return nil, err
	}
	if len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}

	rtCfgs, err := dotnet.RuntimeConfigJSONFiles(".")
	if err != nil {
		return nil, fmt.Errorf("finding runtimeconfig.json: %w", err)
	}
	if len(rtCfgs) > 0 {
		return gcp.OptIn("found at least one runtimeconfig.json"), nil
	}

	return gcp.OptOut("no project files or .dll files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	rtCfgFiles, err := dotnet.RuntimeConfigJSONFiles(".")
	if err != nil {
		return fmt.Errorf("finding runtimeconfig.json: %w", err)
	}
	// This invalid state should have been rejected by the detectFn already.
	if len(rtCfgFiles) == 0 {
		return fmt.Errorf("runtimeconfig.json does not exist")
	}

	runtimeVersion, err := dotnet.GetRuntimeVersion(ctx, rtCfgFiles)
	if err != nil {
		return err
	}
	isDevMode, err := env.IsDevMode()
	if err != nil {
		return fmt.Errorf("checking if dev mode is enabled: %w", err)
	}
	if !isDevMode {
		if err := buildRuntimeLayer(ctx, runtimeVersion); err != nil {
			return fmt.Errorf("building the runtime layer: %w", err)
		}
	}
	return nil
}

func buildRuntimeLayer(ctx *gcp.Context, rtVersion string) error {
	rtl, err := ctx.Layer(runtimeLayerName, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", runtimeLayerName, err)
	}
	if _, err := runtime.InstallTarballIfNotCached(ctx, runtime.AspNetCore, rtVersion, rtl); err != nil {
		return err
	}
	ctx.AddInstalledRuntimeVersion(rtVersion)
	rtl.LaunchEnvironment.Default("DOTNET_ROOT", rtl.Path)
	rtl.LaunchEnvironment.Prepend("PATH", string(os.PathListSeparator), rtl.Path)
	rtl.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")
	return nil
}
