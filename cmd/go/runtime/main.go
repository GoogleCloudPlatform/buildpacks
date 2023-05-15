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

// Implements go/runtime buildpack.
// The runtime buildpack installs the Go toolchain.
package main

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	goLayer      = "go"
	envGoVersion = "GOOGLE_GO_VERSION"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("go"); result != nil {
		return result, nil
	}
	atLeastOne, err := ctx.HasAtLeastOneOutsideDependencyDirectories("*.go")
	if err != nil {
		return nil, fmt.Errorf("finding *.go files: %w", err)
	}
	if atLeastOne {
		return gcp.OptIn("found .go files"), nil
	}
	return gcp.OptOut("no .go files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}
	grl, err := ctx.Layer(goLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Go, version, grl)
	return err
}

func runtimeVersion(ctx *gcp.Context) (string, error) {
	if version := os.Getenv(envGoVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", envGoVersion, version)
		return version, nil
	}

	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	ctx.Logf("Using latest stable Go version")
	return "", nil
}
