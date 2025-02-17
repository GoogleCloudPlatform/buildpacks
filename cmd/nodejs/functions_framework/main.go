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

// Implements nodejs/functions_framework buildpack.
// The functions_framework buildpack converts a function into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName                 = "functions-framework"
	functionsFrameworkPackage = "@google-cloud/functions-framework"

	// nodeJSHeadroomMB is the amount of memory we'll set aside before computing the max memory size.
	nodeJSHeadroomMB int = 64
)

var functionsFrameworkNodeModulePath = path.Join("node_modules", functionsFrameworkPackage)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if nodejs.IsNodeJS8Runtime() {
		return gcp.OptOut("Incompatible with nodejs8"), nil
	}
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

// buildFn sets up the execution environment for the function.
// For a function that specifies the framework as a dependency, only set
// environment variables and define a web process. The framework is
// installed in the npm or yarn buildpack with other dependencies.
// For a function that does not, also install the framework.
func buildFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionSource); ok {
		return gcp.UserErrorf("%s is not currently supported for Node.js buildpacks", env.FunctionSource)
	}

	indexJSExists, err := ctx.FileExists("index.js")
	if err != nil {
		return err
	}
	// Function source code should be defined in the "main" field in package.json, index.js or function.js.
	// https://cloud.google.com/functions/docs/writing#structuring_source_code
	fnFile := "function.js"
	if indexJSExists {
		fnFile = "index.js"
	}

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}
	if pjs != nil {
		_, hasFrameworkDependency = pjs.Dependencies[functionsFrameworkPackage]
		if pjs.Main != "" {
			fnFile = pjs.Main
		}
	}

	fnFileExists, err := ctx.FileExists(fnFile)
	if err != nil {
		return err
	}
	if !fnFileExists {
		return gcp.UserErrorf("%s does not exist", fnFile)
	}

	yarnPnP, err := usingYarnModuleResolution(ctx)
	if err != nil {
		return err
	}

	if yarnPnP && !hasFrameworkDependency {
		return gcp.UserErrorf("This project is using Yarn Plug'n'Play but you have not included the Functions Framework in your dependencies. Please add it by running: 'yarn add @google-cloud/functions-framework'.")
	}

	pnpmLockExists, err := ctx.FileExists(nodejs.PNPMLock)
	if err != nil {
		return err
	}
	if pnpmLockExists && !hasFrameworkDependency {
		return gcp.UserErrorf("This project is using pnpm but you have not included the Functions Framework in your dependencies. Please add it by running: 'pnpm add @google-cloud/functions-framework'.")
	}

	// TODO(mattrobertson) remove this check once Nodejs has backported the fix to v16. More info here:
	// https://github.com/GoogleCloudPlatform/functions-framework-nodejs/issues/407
	if skip, err := nodejs.SkipSyntaxCheck(ctx, fnFile, pjs); err != nil {
		return err
	} else if !skip {
		// Syntax check the function code without executing to prevent run-time errors.
		if yarnPnP {
			if _, err := ctx.Exec([]string{"yarn", "node", "--check", fnFile}, gcp.WithUserAttribution); err != nil {
				return err
			}
		} else {
			if _, err := ctx.Exec([]string{"node", "--check", fnFile}, gcp.WithUserAttribution); err != nil {
				return err
			}
		}
	}

	l, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}
	// We use the absolute path to the functions-framework executable in order to
	// avoid having to add its parent directory to PATH which could cause
	// conflicts with user-specified dependencies in the case where the framework
	// is not an explicit dependency.
	//
	// If the function specifies a framework dependency, the executable will be
	// in node_modules, where it would have been installed by the preceding
	// npm/yarn buildpack. Otherwise, it will be in the layer's node_modules,
	// installed below.
	ff := filepath.Join(".bin", "functions-framework")

	if yarnPnP {
		// In order for node module resolution to work in Yarn Plug'n'Play mode, we must invoke yarn to
		// start the Functions Framework.
		ff = "yarn functions-framework"
		cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
			Runtime: "nodejs",
			Version: "yarn",
		})
	} else if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		if err := ctx.ClearLayer(l); err != nil {
			return fmt.Errorf("clearing layer %q: %w", l.Name, err)
		}
		ff = filepath.Join("node_modules", ff)
		addFrameworkVersionLabel(ctx, functionsFrameworkNodeModulePath, false)
	} else {
		ctx.Logf("Handling functions without dependency on functions-framework.")
		if err := cloudfunctions.AssertFrameworkInjectionAllowed(); err != nil {
			return err
		}

		if err := installFunctionsFramework(ctx, l); err != nil {
			vendorError := ""
			if nodejs.IsUsingVendoredDependencies() {
				vendorError = "Vendored dependencies detected, please make sure you have functions-framework installed locally to avoid the installation error by following: https://github.com/GoogleCloudPlatform/functions-framework-nodejs#installation."
			}

			return fmt.Errorf("%s installing functions-framework: %w", vendorError, err)
		}

		ff = filepath.Join(l.Path, "node_modules", ff)
		addFrameworkVersionLabel(ctx, filepath.Join(l.Path, functionsFrameworkNodeModulePath), true)

		nm := filepath.Join(ctx.ApplicationRoot(), "node_modules")
		nmExists, err := ctx.FileExists(nm)
		if err != nil {
			return err
		}
		// Add user's node_modules to NODE_PATH so functions-framework can always find user's packages.
		if nmExists {
			l.LaunchEnvironment.Prepend("NODE_PATH", string(os.PathListSeparator), nm)
		}
	}

	// Get and set the valid value for --max-old-space-size node_options.
	// Keep the existing behaviour if the value is not provided or invalid
	if size, err := getMaxOldSpaceSize(); err != nil {
		return err
	} else if size > 0 {
		l.LaunchEnvironment.Prepend("NODE_OPTIONS", " ", fmt.Sprintf("--max-old-space-size=%d", size))
	}

	if err := ctx.SetFunctionsEnvVars(l); err != nil {
		return err
	}
	ctx.AddWebProcess([]string{"/bin/bash", "-c", ff})
	return nil
}

// installFunctionsFramework downloads the functions-framework package to node_modules in the given layer.
func installFunctionsFramework(ctx *gcp.Context, l *libcnb.Layer) error {
	nodeVersion := os.Getenv(env.Runtime)
	var subdir string
	if nodeVersion == "nodejs12" || nodeVersion == "nodejs14" {
		subdir = "without-framework-compat"
	} else {
		subdir = "without-framework"
	}

	cvt := filepath.Join(ctx.BuildpackRoot(), "converter", subdir)
	pjs := filepath.Join(cvt, "package.json")
	pljs := filepath.Join(cvt, nodejs.PackageLock)

	cached, err := nodejs.CheckOrClearCache(ctx, l, cache.WithStrings(nodejs.EnvProduction), cache.WithFiles(pjs, pljs))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		return nil
	}
	installCmd, err := nodejs.NPMInstallCommand(ctx)
	if err != nil {
		return err
	}
	// NPM expects package.json and the lock file in the prefix directory.
	if _, err := ctx.Exec([]string{"cp", "-t", l.Path, pjs, pljs}, gcp.WithUserTimingAttribution); err != nil {
		return err
	}
	if err := ar.GenerateNPMConfig(ctx); err != nil {
		return fmt.Errorf("generating Artifact Registry credentials: %w", err)
	}
	if _, err := ctx.Exec([]string{"npm", installCmd, "--quiet", "--production", "--prefix", l.Path}, gcp.WithUserAttribution); err != nil {
		return err
	}
	return nil
}

// getMaxOldSpaceSize returns the memory size specified by (GOOGLE_CONTAINER_MEMORY_HINT_MB - nodeJSHeadroomMB),
// or 0 if env var is not specified.
func getMaxOldSpaceSize() (int, error) {
	memHintStr, exist := os.LookupEnv(env.ContainerMemoryHintMB)
	if !exist {
		return 0, nil
	}

	memHint, err := strconv.Atoi(memHintStr)
	if err != nil {
		return 0, fmt.Errorf("%s=%q must be an integer: %v", env.ContainerMemoryHintMB, memHintStr, err)
	}

	if memHint <= nodeJSHeadroomMB {
		return 0, fmt.Errorf("%s=%q must be greater than %d", env.ContainerMemoryHintMB, memHintStr, nodeJSHeadroomMB)
	}

	return memHint - nodeJSHeadroomMB, nil
}

// tryAddFrameworkVersionLabel attempts to identify the functions framework
// version being used by reading the functions-framework package's manifest.
// If the version is detected it is added to the generated image.
func addFrameworkVersionLabel(ctx *gcp.Context, ffPackageJSON string, injected bool) {
	version := "unknown"
	packageInfo, err := nodejs.ReadPackageJSONIfExists(ffPackageJSON)
	if err != nil {
		ctx.Logf("Could not detect installed functions framework version: %v", err)
	} else {
		version = packageInfo.Version
	}
	cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
		Runtime:  "nodejs",
		Version:  version,
		Injected: injected,
	})
}

// usingYarnModuleResolution returns true if this project was built using a new version of Yarn that
// does not create the "node_modules" directory.
func usingYarnModuleResolution(ctx *gcp.Context) (bool, error) {
	yarnLockExists, err := ctx.FileExists(nodejs.YarnLock)
	if err != nil {
		return false, err
	}
	if !yarnLockExists {
		return false, nil
	}
	yarn2, err := nodejs.IsYarn2(ctx.ApplicationRoot())
	if err != nil || !yarn2 {
		return false, nil
	}
	result, err := ctx.Exec([]string{"yarn", "config", "get", "nodeLinker"}, gcp.WithUserAttribution)
	if err != nil {
		return false, err
	}
	linker := result.Stdout
	return linker == "pnp", nil
}
