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
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb"
)

const (
	layerName                 = "functions-framework"
	functionsFrameworkPackage = "@google-cloud/functions-framework"
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

// buildFn sets up the execution environment for the function.
// For a function that specifies the framework as a dependency, only set
// environment variables and define a web process. The framework is
// installed in the npm or yarn buildpack with other dependencies.
// For a function that does not, also install the framework.
func buildFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionSource); ok {
		return gcp.UserErrorf("%s is not currently supported for Node.js buildpacks", env.FunctionSource)
	}

	// Function source code should be defined in the "main" field in package.json, index.js or function.js.
	// https://cloud.google.com/functions/docs/writing#structuring_source_code
	fnFile := "function.js"
	if ctx.FileExists("index.js") {
		fnFile = "index.js"
	}

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	if ctx.FileExists("package.json") {
		pjs, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
		if err != nil {
			return fmt.Errorf("reading package.json: %w", err)
		}
		_, hasFrameworkDependency = pjs.Dependencies[functionsFrameworkPackage]
		if pjs.Main != "" {
			fnFile = pjs.Main
		}
	}

	if !ctx.FileExists(fnFile) {
		return gcp.UserErrorf("%s does not exist", fnFile)
	}

	// Syntax check the function code without executing to prevent run-time errors.
	ctx.Exec([]string{"node", "--check", fnFile}, gcp.WithUserAttribution)

	l := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
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

	if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		ctx.ClearLayer(l)
		ff = filepath.Join("node_modules", ff)
	} else {
		ctx.Logf("Handling functions without dependency on functions-framework.")

		if err := installFunctionsFramework(ctx, l); err != nil {
			return fmt.Errorf("installing funtions-framework: %w", err)
		}

		ff = filepath.Join(l.Path, "node_modules", ff)

		// Add user's node_modules to NODE_PATH so functions-framework can always find user's packages.
		if nm := filepath.Join(ctx.ApplicationRoot(), "node_modules"); ctx.FileExists(nm) {
			l.LaunchEnvironment.Prepend("NODE_PATH", string(os.PathListSeparator), nm)
		}
	}

	ctx.SetFunctionsEnvVars(l)
	ctx.AddWebProcess([]string{"/bin/bash", "-c", ff})
	return nil
}

// installFunctionsFramework downloads the functions-framework package to node_modules in the given layer.
func installFunctionsFramework(ctx *gcp.Context, l *libcnb.Layer) error {
	cvt := filepath.Join(ctx.BuildpackRoot(), "converter", "without-framework")
	pjs := filepath.Join(cvt, "package.json")
	pljs := filepath.Join(cvt, nodejs.PackageLock)

	cached, err := nodejs.CheckCache(ctx, l, cache.WithStrings(nodejs.EnvProduction), cache.WithFiles(pjs, pljs))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(layerName)
		return nil
	}
	installCmd, err := nodejs.NPMInstallCommand(ctx)
	if err != nil {
		return err
	}

	ctx.CacheMiss(layerName)
	ctx.ClearLayer(l)
	// NPM expects package.json and the lock file in the prefix directory.
	ctx.Exec([]string{"cp", "-t", l.Path, pjs, pljs}, gcp.WithUserTimingAttribution)
	ctx.Exec([]string{"npm", installCmd, "--quiet", "--production", "--prefix", l.Path}, gcp.WithUserAttribution)
	return nil
}
