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

// Implements php/runtime buildpack.
// The runtime buildpack installs the PHP runtime.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

// usePHPRuntime is used to enable the php/runtime buildpack
const usePHPRuntime = "GOOGLE_USE_EXPERIMENTAL_PHP_RUNTIME"

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	isEnabled, err := env.IsPresentAndTrue(usePHPRuntime)
	if err != nil {
		ctx.Warnf("failed to parse %s: %v", usePHPRuntime, err)
	}

	if !isEnabled {
		return gcp.OptOutEnvNotSet(usePHPRuntime), nil
	}

	if result := runtime.CheckOverride(ctx, "php"); result != nil {
		return result, nil
	}

	if ctx.FileExists("composer.json") {
		return gcp.OptInFileFound("composer.json"), nil
	}
	if ctx.HasAtLeastOne("*.php") {
		return gcp.OptIn(".php files found"), nil
	}
	return gcp.OptOut("composer.json or .php files not found"), nil

}

func buildFn(ctx *gcp.Context) error {
	version, err := php.ExtractVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	phpl := ctx.Layer("php", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.PHP, version, phpl)
	return err
}
