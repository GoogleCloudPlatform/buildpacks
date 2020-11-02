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
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {

	if err := validateSource(ctx); err != nil {
		return err
	}

	// Check for syntax errors.
	ctx.Exec([]string{"python3", "-m", "compileall", "-f", "-q", "."}, gcp.WithStdoutTail, gcp.WithUserAttribution)

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	if ctx.FileExists("requirements.txt") {
		content := ctx.ReadFile("requirements.txt")
		hasFrameworkDependency = containsFF(string(content))
	}

	// Install functions-framework.
	l := ctx.Layer(layerName, gcp.LaunchLayer, gcp.BuildLayer)
	if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		ctx.ClearLayer(l)
	} else {
		ctx.Logf("Handling functions without dependency on functions-framework.")
		cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
		req := filepath.Join(cvt, "requirements.txt")
		if _, err := python.InstallRequirements(ctx, l, req); err != nil {
			return fmt.Errorf("installing framework: %w", err)
		}
	}

	ctx.SetFunctionsEnvVars(l)
	ctx.AddWebProcess([]string{"functions-framework"})
	return nil
}

func validateSource(ctx *gcp.Context) error {
	// Fail if the default|custom source file doesn't exist, otherwise the app will fail at runtime but still build here.
	fnSource, ok := os.LookupEnv(env.FunctionSource)
	if !ok {
		if !ctx.FileExists("main.py") {
			return gcp.UserErrorf("missing main.py and %s not specified. Either create the function in main.py or specify %s to point to the file that contains the function", env.FunctionSource, env.FunctionSource)
		}
	} else if !ctx.FileExists(fnSource) {
		return gcp.UserErrorf("%s specified file '%s' but it does not exist", env.FunctionSource, fnSource)
	}
	return nil
}

func containsFF(s string) bool {
	return ffRegexp.MatchString(s) || eggRegexp.MatchString(s)
}
