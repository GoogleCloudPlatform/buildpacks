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
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

const (
	layerName = "functions-framework-compat"
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
	// Install Python 3.7 default dependencies for compatibility with GCF.
	lc := ctx.Layer(layerName, gcp.LaunchLayer)
	cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
	req := filepath.Join(cvt, "requirements.txt")
	if err := python.InstallRequirements(ctx, lc, req); err != nil {
		return err
	}
	// Set additional Python 3.7 env var.
	lc.LaunchEnvironment.Default("ENTRY_POINT", os.Getenv(env.FunctionTarget))

	return nil
}
