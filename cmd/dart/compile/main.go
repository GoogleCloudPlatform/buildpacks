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

// Implements dart/compile buildpack.
// The compile buildpack runs dart compile to produce a self-contained executable.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	atLeastOne, err := ctx.HasAtLeastOne("*.dart")
	if err != nil {
		return nil, fmt.Errorf("finding *.dart files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .dart files found"), nil
	}
	return gcp.OptIn("found .dart files"), nil
}

func buildFn(ctx *gcp.Context) error {
	// Create a layer for the compiled binary.  Add it to PATH in case
	// users wish to invoke the binary manually.
	bl, err := ctx.Layer("bin", gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	bl.LaunchEnvironment.Prepend("PATH", string(os.PathListSeparator), bl.Path)
	outBin := filepath.Join(bl.Path, "server")

	buildable, err := dartBuildable(ctx)
	if err != nil {
		return fmt.Errorf("unable to find a valid buildable: %w", err)
	}

	// Build the application.
	bld := []string{"dart", "compile", "exe", buildable, "-o", outBin}
	ctx.Exec(bld, gcp.WithUserAttribution)

	ctx.AddWebProcess([]string{"/bin/bash", "-c", outBin})
	return nil
}

func dartBuildable(ctx *gcp.Context) (string, error) {

	// The user tells us what to build.
	if buildable, ok := os.LookupEnv(env.Buildable); ok {
		return buildable, nil
	}

	// Default to bin/server.dart in the application root.
	return "bin/server.dart", nil
}
