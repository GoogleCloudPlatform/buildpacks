// Copyright 2023 Google LLC
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

// Implements dotnet/flex buildpack.
// The flex buildpack sets appropriate envars for dotnet on GAE Flex.
// For details, see this page on .NET 6:
// https://learn.microsoft.com/en-us/aspnet/core/fundamentals/configuration/?view=aspnetcore-6.0
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	// UrlsEnvar is the name of the dotnet-specific envar to set application URL(s), including port.
	UrlsEnvar = "ASPNETCORE_URLS"
)

func main() {
	gcpbuildpack.Main(detectFn, buildFn)
}

func detectFn(ctx *gcpbuildpack.Context) (gcpbuildpack.DetectResult, error) {
	if !env.IsFlex() {
		return gcpbuildpack.OptOut("not a GAE Flex app."), nil
	}
	return gcpbuildpack.OptIn("this is a GAE Flex app."), nil
}

func buildFn(ctx *gcpbuildpack.Context) error {
	l, err := ctx.Layer("main_env", gcpbuildpack.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating main_env layer: %v", err)
	}
	l.LaunchEnvironment.Default(UrlsEnvar, "http://0.0.0.0:8080")
	return nil
}
