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
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	phpIniName = "php.ini"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("php"); result != nil {
		return result, nil
	}

	composerJSONExists, err := ctx.FileExists("composer.json")
	if err != nil {
		return nil, err
	}
	if composerJSONExists {
		return gcp.OptInFileFound("composer.json"), nil
	}
	atLeastOne, err := ctx.HasAtLeastOne("*.php")
	if err != nil {
		return nil, fmt.Errorf("finding *.php files: %w", err)
	}
	if atLeastOne {
		return gcp.OptIn(".php files found"), nil
	}
	return gcp.OptOut("composer.json or .php files not found"), nil

}

func buildFn(ctx *gcp.Context) error {
	version, err := php.ExtractVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	phpl, err := ctx.Layer("php", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)

	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.PHP, version, phpl)
	if err != nil {
		return err
	}

	setPeclConfig(phpl)
	return addPHPIni(ctx, phpl)
}

func setPeclConfig(phpl *libcnb.Layer) {
	phpl.SharedEnvironment.Default("PHP_PEAR_PHP_BIN", filepath.Join(phpl.Path, "bin", "php"))
	phpl.SharedEnvironment.Default("PHP_PEAR_INSTALL_DIR", filepath.Join(phpl.Path, "lib", "php"))
}

func addPHPIni(ctx *gcp.Context, phpl *libcnb.Layer) error {
	destDir := filepath.Join(phpl.Path, "etc")
	destPath := filepath.Join(destDir, phpIniName)

	if err := ctx.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating etc folder: %w", err)
	}

	if err := ctx.WriteFile(destPath, []byte(php.PHPIni), os.FileMode(0755)); err != nil {
		return err
	}

	// PHP uses PHPRC env var to find php.ini
	phpl.LaunchEnvironment.Default("PHPRC", destDir)
	return nil
}
