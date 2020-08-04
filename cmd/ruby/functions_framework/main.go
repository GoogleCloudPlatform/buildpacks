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

// Implements ruby/functions_framework buildpack.
// The functions_framework buildpack sets up the execution environment to
// run the Ruby Functions Framework. The framework itself, with its converter,
// is always installed as a dependency.
package main

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	layerName = "functions-framework"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		ctx.OptIn("%s set", env.FunctionTarget)
	}
	// TODO(b/154846199): For compatibility with GCF; this will be removed later.
	if os.Getenv("CNB_STACK_ID") != "google" {
		if _, ok := os.LookupEnv(env.FunctionTargetLaunch); ok {
			ctx.OptIn("%s set", env.FunctionTargetLaunch)
		}
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	if err := validateSource(ctx); err != nil {
		return err
	}

	// The framework has been installed with the dependencies, so this layer is
	// used only for env vars.
	l := ctx.Layer(layerName, gcp.LaunchLayer)
	ctx.SetFunctionsEnvVars(l)

	// Verify that the framework is installed and ready.
	// TODO(b/156038129): Implement a --verify flag in the functions framework
	// that also checks the actual function for readiness.
	cmd := []string{"bundle", "exec", "functions-framework", "--help"}
	if _, err := ctx.ExecWithErr(cmd, gcp.WithUserAttribution); err != nil {
		return gcp.UserErrorf("unable to execute functions-framework; please ensure the functions_framework gem is in your Gemfile")
	}

	ctx.AddWebProcess([]string{"bundle", "exec", "functions-framework"})

	return nil
}

func validateSource(ctx *gcp.Context) error {
	// Fail if the default|custom source file doesn't exist, otherwise the app will fail at runtime but still build here.
	fnSource, ok := os.LookupEnv(env.FunctionSource)
	if ok && !ctx.FileExists(fnSource) {
		return gcp.UserErrorf("%s specified file '%s' but it does not exist", env.FunctionSource, fnSource)
	}
	return nil
}
