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

// Implements java/clear_source buildpack.
// The clear_source buildpack clears the source out.
package main

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/clearsource"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if err := clearsource.DetectFn(ctx); err != nil {
		return err
	}
	if !ctx.FileExists("pom.xml") && !ctx.FileExists("build.gradle") && !ctx.FileExists("build.gradle.kts") {
		ctx.OptOut("None of pom.xml, build.gradle, nor build.gradle.kts found. Clearing souce only supported on maven and gradle projects.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	return clearsource.BuildFn(ctx, []string{"target", "build"})
}
