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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
)

const (
	// ffPackage is the name of the functions framework Packagist package. It's also the path
	// to the functions framework under the vendor directory, so it's used in both senses.
	ffPackage = "google/cloud-functions-framework"

	// ffVersion is the default version of functions framework to install in the container.
	// This value must match the version specified by converter/composer.json
	ffVersion = "^1.1"

	// ffPackageWithVersion is the package that we `composer require` when adding the functions
	// framework to an existing vendor directory.
	ffPackageWithVersion = ffPackage + ":" + ffVersion

	ffGitHubURL    = "https://github.com/GoogleCloudPlatform/functions-framework-php"
	ffPackagistURL = "https://packagist.org/packages/google/cloud-functions-framework"

	// routerScript is the path to the functions framework invoker script.
	routerScript = "vendor/google/cloud-functions-framework/router.php"

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

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	fnFile := "index.php"
	if fnSource, ok := os.LookupEnv(env.FunctionSource); ok {
		fnFile = fnSource
	}

	// Syntax check the function code without executing.
	command := []string{"php", "-l", fnFile}
	if _, err := ctx.Exec(command, gcp.WithCombinedTail, gcp.WithUserAttribution); err != nil {
		return err
	}

	composerJSONExists, err := ctx.FileExists("composer.json")
	if err != nil {
		return err
	}
	// Install the functions framework if need be.
	if composerJSONExists {
		if err := handleComposerJSON(ctx); err != nil {
			return err
		}
	} else {
		if err := handleNoComposerJSON(ctx); err != nil {
			return err
		}
	}

	ctx.AddWebProcess([]string{"/bin/bash", "-c", fmt.Sprintf("php -S 0.0.0.0:${PORT} %s", routerScript)})

	l, err := ctx.Layer("functions-framework", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	if err := ctx.SetFunctionsEnvVars(l); err != nil {
		return err
	}
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
		if err := cloudfunctions.AssertFrameworkInjectionAllowed(); err != nil {
			return err
		}
		if err := php.ComposerRequire(ctx, []string{ffPackageWithVersion}); err != nil {
			return err
		}
		cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
			Runtime:  "php",
			Version:  ffVersion,
			Injected: true,
		})
	} else {
		ctx.Logf("Handling function with dependency on functions framework (%s:%s)", ffPackage, version)
		cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
			Runtime:  "php",
			Version:  version,
			Injected: false,
		})
	}

	return nil
}

// handleNoComposerJSON installs the functions framework, if required, in the case
// that there is no composer.json file present.
func handleNoComposerJSON(ctx *gcp.Context) error {
	ctx.Logf("Handling function without composer.json")

	vendorExists, err := ctx.FileExists(php.Vendor)
	if err != nil {
		return err
	}
	// Check if there's a vendor directory. If not, this is truly a dependency-less function
	// so we can `composer install` the framework and cache the vendor dir.
	if !vendorExists {
		ctx.Logf("No vendor directory present, installing functions framework")
		cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
		if _, err := ctx.Exec([]string{"cp", filepath.Join(cvt, "composer.json"), filepath.Join(cvt, "composer.lock"), "."}); err != nil {
			return err
		}

		if _, err := php.ComposerInstall(ctx, cacheTag); err != nil {
			return fmt.Errorf("composer install: %w", err)
		}

		cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
			Runtime:  "php",
			Version:  ffVersion,
			Injected: true,
		})

		return nil
	}

	ffPath := filepath.Join(php.Vendor, ffPackage)
	ffExists, err := ctx.FileExists(ffPath)
	if err != nil {
		return err
	}
	// Check if the vendor directory contains the functions framework. If so we're done.
	if ffExists {
		ctx.Logf("Functions framework is already present in the vendor directory")

		routerScriptExists, err := ctx.FileExists(routerScript)
		if err != nil {
			return err
		}
		// Make sure the router script also exists. If the user is vendoring their own deps
		// you never know how they've structured their vendor directory.
		if !routerScriptExists {
			return gcp.UserErrorf("functions framework router script %s is not present", routerScript)
		}

		cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
			Runtime:  "php",
			Version:  "unknown-vendored",
			Injected: false,
		})

		return nil
	}

	if err := cloudfunctions.AssertFrameworkInjectionAllowed(); err != nil {
		return err
	}

	// The user did not vendor the functions framework. Before installing it, let's see if they used
	// Composer to install their deps. If so we can safely `composer require` the framework even
	// without composer.json; vendor/composer/installed.json contains the info required to resolve
	// a working set of dependencies.
	ctx.Warnf("Functions framework is not present at %s, so automatic injection will be attempted. Please add a dependency on it to avoid unexpected conflicts or breakages that result from this. See %s and %s", ffPath, ffGitHubURL, ffPackagistURL)
	installed := filepath.Join(php.Vendor, "composer", "installed.json")
	installedExists, err := ctx.FileExists(installed)
	if err != nil {
		return err
	}
	if !installedExists {
		return gcp.UserErrorf("%s is not present, so it appears that Composer was not used to install dependencies.", installed)
	}

	// All clear to install the functions framework! We'll do this via `composer require`
	// because we're adding a package to an already existing vendor directory.
	ctx.Logf("Installing functions framework %s", ffPackageWithVersion)
	if err := php.ComposerRequire(ctx, []string{ffPackageWithVersion}); err != nil {
		return nil
	}

	cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
		Runtime:  "php",
		Version:  ffVersion,
		Injected: true,
	})

	return nil
}
