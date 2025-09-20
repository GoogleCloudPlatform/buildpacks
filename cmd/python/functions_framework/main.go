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

// Implements python/functions_framework buildpack.
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

const (
	layerName = "functions-framework"
)

var (
	functionsFramework = "functions-framework"
	requirementsTxt    = "requirements.txt"
	pyprojectToml      = "pyproject.toml"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		if python.IsPyprojectEnabled(ctx) {
			return gcp.OptInEnvSet(env.FunctionTarget), nil
		}
		return gcp.OptInEnvSet(env.FunctionTarget, gcp.WithBuildPlans(python.RequirementsProvidesPlan)), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	if err := validateSource(ctx); err != nil {
		return err
	}

	// Check for syntax errors to prevent failures that would only manifest at run time.
	if _, err := ctx.Exec([]string{"python3", "-m", "compileall", "-f", "-q", "."}, gcp.WithStdoutTail, gcp.WithUserAttribution); err != nil {
		return err
	}

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	pyprojectTomlExists, err := ctx.FileExists(pyprojectToml)
	if err != nil {
		return err
	}

	hasFrameworkDependency, err = python.PackagePresent(ctx, functionsFramework)
	if err != nil {
		return err
	}

	// Install functions-framework if necessary.
	l, err := ctx.Layer(layerName, gcp.LaunchLayer, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}
	if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		if err := ctx.ClearLayer(l); err != nil {
			return fmt.Errorf("clearing layer %q: %w", l.Name, err)
		}
	} else {
		if pyprojectTomlExists && env.IsAlphaSupported() {
			return gcp.UserErrorf("This project is using pyproject.toml but you have not included the Functions Framework in your dependencies. Please add it by running: 'poetry add functions-framework' or 'uv add functions-framework'.")
		}

		if _, isVendored := os.LookupEnv(python.VendorPipDepsEnv); isVendored {
			return gcp.UserErrorf("Vendored dependencies detected, please add functions-framework to requirements.txt and download it using pip")
		}
		ctx.Logf("Handling functions without dependency on functions-framework.")
		if err := cloudfunctions.AssertFrameworkInjectionAllowed(); err != nil {
			return err
		}

		// The pip install is performed by the pip buildpack; see python.InstallRequirements.
		ctx.Logf("Adding functions-framework requirements.txt to the list of requirements files to install.")
		r := filepath.Join(ctx.BuildpackRoot(), "converter", "requirements.txt")
		l.BuildEnvironment.Append(python.RequirementsFilesEnv, string(os.PathListSeparator), r)
	}

	if err := ctx.SetFunctionsEnvVars(l); err != nil {
		return err
	}
	ctx.AddWebProcess([]string{"functions-framework"})
	return nil
}

func validateSource(ctx *gcp.Context) error {
	// Fail if the default|custom source file doesn't exist, otherwise the app will fail at runtime but still build here.
	fnSource, ok := os.LookupEnv(env.FunctionSource)
	if !ok {
		mainPYExists, err := ctx.FileExists("main.py")
		if err != nil {
			return err
		}
		if !mainPYExists {
			return gcp.UserErrorf("missing main.py and %s not specified. Either create the function in main.py or specify %s to point to the file that contains the function", env.FunctionSource, env.FunctionSource)
		}
	} else {
		fnSourceExists, err := ctx.FileExists(fnSource)
		if err != nil {
			return err
		}
		if !fnSourceExists {
			return gcp.UserErrorf("%s specified file %q but it does not exist", env.FunctionSource, fnSource)
		}
	}
	return nil
}
