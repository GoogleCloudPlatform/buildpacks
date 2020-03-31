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

// Implements dotnet/appengine_main buildpack.
// The appengine_main buildpack handles the app.yaml `main` field when specified.
package main

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if proj := os.Getenv(env.GAEMain); proj == "" {
		ctx.OptOut("app.yaml main field is not defined, using default")
	}

	if _, exists := os.LookupEnv(env.Buildable); exists {
		ctx.OptOut("%s is set, ignoring app.yaml main field", env.Buildable)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("main_env")
	ctx.OverrideBuildEnv(l, env.Buildable, os.Getenv(env.GAEMain))
	ctx.WriteMetadata(l, nil, layers.Build)
	return nil
}
