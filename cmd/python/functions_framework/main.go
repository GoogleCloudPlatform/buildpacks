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
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

const (
	layerName = "functions-framework"
)

var (
	ffRegexp  = regexp.MustCompile(`(?m)^functions-framework\b([^-]|$)`)
	eggRegexp = regexp.MustCompile(`(?m)#egg=functions-framework$`)
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget, gcp.WithBuildPlans(python.RequirementsProvidesPlan)), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := validateSource(ctx); err != nil {
		return err
	}

	// Check for syntax errors to prevent failures that would only manifest at run time.
	if _, err := ctx.Exec([]string{"python3", "-m", "compileall", "-f", "-q", "."}, gcp.WithStdoutTail, gcp.WithUserAttribution); err != nil {
		return err
	}

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	requirementsExists, err := ctx.FileExists("requirements.txt")
	if err != nil {
		return err
	}
	if requirementsExists {
		content, err := ctx.ReadFile("requirements.txt")
		if err != nil {
			return err
		}
		hasFrameworkDependency = containsFF(string(content))
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
		ctx.Logf("Handling functions without dependency on functions-framework.")
		if err := cloudfunctions.AssertFrameworkInjectionAllowed(); err != nil {
			return err
		}

		// The pip install is performed by the pip buildpack; see python.InstallRequirements.
		ctx.Debugf("Adding functions-framework requirements.txt to the list of requirements files to install.")
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

func containsFF(s string) bool {
	return ffRegexp.MatchString(s) || eggRegexp.MatchString(s)
}
