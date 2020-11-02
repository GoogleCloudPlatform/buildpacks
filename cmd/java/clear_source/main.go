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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/clearsource"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result, err := clearsource.DetectFn(ctx); result != nil || err != nil {
		return result, err
	}

	files := []string{
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
	}
	for _, f := range files {
		if ctx.FileExists(f) {
			return gcp.OptInFileFound(f), nil
		}
	}
	return gcp.OptOut(fmt.Sprintf("none of %s found. Clearing souce only supported on maven and gradle projects.", strings.Join(files, ", "))), nil
}

func buildFn(ctx *gcp.Context) error {
	return clearsource.BuildFn(ctx, []string{"target", "build"})
}
