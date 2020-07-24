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

// Implements php/functions_framework buildpack.
// The functions_framework buildpack converts a function into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	// ffPackage is the name of the functions framework Packagist package. It's also the path
	// to the functions framework under the vendor directory, so it's used in both senses.
	ffPackage = "google/cloud-functions-framework"

	// ffPackageWithVersion is the package that we `composer require` when adding the functions
	// framework to an existing vendor directory.
	ffPackageWithVersion = ffPackage + ":^0.3"

	ffGitHubURL    = "https://github.com/GoogleCloudPlatform/functions-framework-php"
	ffPackagistURL = "https://packagist.org/packages/google/cloud-functions-framework"

	// routerScript is the path to the functions framework invoker script.
	routerScript = "vendor/bin/router.php"

	// cacheTag is the cache tag for the `composer install` layer. We only cache in one case: There
	// is no composer.json file and there is no vendor directory (i.e. a dependency-less function).
	// That's the only case where we create the vendor dir from scratch, so it's cacheable based on
	// the composer.lock file. Other cases involve modifying an existing vendor directory, whether
	// created by the composer buildpack or provided by the user.
	cacheTag = "functions-framework dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		ctx.OptIn("%s set", env.FunctionTarget)
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	fnFile := "index.php"
	if fnSource, ok := os.LookupEnv(env.FunctionSource); ok {
		fnFile = fnSource
	}

	// Syntax check the function code without executing.
	command := []string{"php", "-l", fnFile}
	ctx.Exec(command, gcp.WithStdoutTail, gcp.WithUserAttribution)

	// Install the functions framework if need be.
	if ctx.FileExists("composer.json") {
		if err := handleComposerJSON(ctx); err != nil {
			return err
		}
	} else {
		if err := handleNoComposerJSON(ctx); err != nil {
			return err
		}
	}

	ctx.AddWebProcess([]string{"/bin/bash", "-c", fmt.Sprintf("php -S 0.0.0.0:${PORT} %s", routerScript)})

	l := ctx.Layer("functions-framework")
	ctx.SetFunctionsEnvVars(l)
	ctx.WriteMetadata(l, nil, layers.Build, layers.Launch)
	return nil
}

// handleComposerJSON installs the functions framework, if required, in the case
// that a composer.json file is present.
func handleComposerJSON(ctx *gcp.Context) error {
	cjs, err := php.ReadComposerJSON(ctx.ApplicationRoot())
	if err != nil {
		return fmt.Errorf("reading composer.json: %w", err)
	}

	// Determine if the function has a dependency on the functions framework.
	if version, ok := cjs.Require[ffPackage]; !ok {
		ctx.Logf("Handling function without dependency on functions framework")
		php.ComposerRequire(ctx, []string{ffPackageWithVersion})
	} else {
		ctx.Logf("Handling function with dependency on functions framework (%s:%s)", ffPackage, version)
	}

	return nil
}

// handleNoComposerJSON installs the functions framework, if required, in the case
// that there is no composer.json file present.
func handleNoComposerJSON(ctx *gcp.Context) error {
	ctx.Logf("Handling function without composer.json")

	// Check if there's a vendor directory. If not, this is truly a dependency-less function
	// so we can `composer install` the framework and cache the vendor dir.
	if !ctx.FileExists(php.Vendor) {
		ctx.Logf("No vendor directory present, installing functions framework")
		cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
		ctx.Exec([]string{"cp", filepath.Join(cvt, "composer.json"), filepath.Join(cvt, "composer.lock"), "."})

		if _, err := php.ComposerInstall(ctx, cacheTag); err != nil {
			return fmt.Errorf("composer install: %w", err)
		}

		return nil
	}

	// Check if the vendor directory contains the functions framework. If so we're done.
	ffPath := filepath.Join(php.Vendor, ffPackage)
	if ctx.FileExists(ffPath) {
		ctx.Logf("Functions framework is already present in the vendor directory")

		// Make sure the router script also exists. If the user is vendoring their own deps
		// you never know how they've structured their vendor directory.
		if !ctx.FileExists(routerScript) {
			return gcp.UserErrorf("functions framework router script %s is not present", routerScript)
		}

		return nil
	}

	// The user did not vendor the functions framework. Before installing it, let's see if they used
	// Composer to install their deps. If so we can safely `composer require` the framework even
	// without composer.json; vendor/composer/installed.json contains the info required to resolve
	// a working set of dependencies.
	ctx.Logf("Functions framework is not present at %s", ffPath)
	if installed := filepath.Join(php.Vendor, "composer", "installed.json"); !ctx.FileExists(installed) {
		return gcp.UserErrorf("%s is not present, so it appears that Composer was not used to install dependencies. "+
			"Please install the functions framework at %s. See %s and %s.", installed, ffPath, ffGitHubURL, ffPackagistURL)
	}

	// All clear to install the functions framework! We'll do this via `composer require`
	// because we're adding a package to an already existing vendor directory.
	ctx.Logf("Installing functions framework %s", ffPackageWithVersion)
	php.ComposerRequire(ctx, []string{ffPackageWithVersion})

	return nil
}
