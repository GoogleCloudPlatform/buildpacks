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

// Implements dotnet/functions_framework buildpack.
// The functions_framework buildpack sets up the execution environment for functions.
package main

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
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
			ctx.OptIn("%s %s set", env.FunctionTargetLaunch, os.Getenv("CNB_STACK_ID"))
		}
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName)
	ctx.SetFunctionsEnvVars(l)
	ctx.WriteMetadata(l, nil, layers.Build, layers.Launch)
	return nil
}
