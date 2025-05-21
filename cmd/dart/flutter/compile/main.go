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

// Implements flutter/compile buildpack.
// The compile buildpack runs flutter to build a webapp and dart to compile the server.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dart"
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
	flutter, err := dart.IsFlutter(ctx.ApplicationRoot())
	if err != nil {
		return nil, err
	}
	if !flutter {
		return gcp.OptOut("pubspec.yaml does not include flutter dependency"), nil
	}
	return gcp.OptIn("found .dart files"), nil
}

func maybeRunBuildRunner(ctx *gcp.Context, dir string) error {
	br, err := dart.HasBuildRunner(dir)
	if err != nil {
		return err
	}
	if br {
		// Run build runner.
		if _, err := ctx.Exec([]string{"dart", "run", "build_runner", "build", "--delete-conflicting-outputs"}, gcp.WithUserAttribution, gcp.WithWorkDir(dir)); err != nil {
			return err
		}
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	rootPubspec, err := dart.GetPubspec(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	server := filepath.Join(ctx.ApplicationRoot(), *rootPubspec.Buildpack.Server)
	static := filepath.Join(ctx.ApplicationRoot(), *rootPubspec.Buildpack.Static)

	if rootPubspec.Buildpack.Prebuild != nil {
		if _, err := ctx.Exec([]string{"sh", "-c", *rootPubspec.Buildpack.Prebuild}, gcp.WithUserAttribution, gcp.WithWorkDir(ctx.ApplicationRoot())); err != nil {
			return err
		}
	}

	err = maybeRunBuildRunner(ctx, server)
	if err != nil {
		return err
	}

	err = maybeRunBuildRunner(ctx, static)
	if err != nil {
		return err
	}

	// Create a layer for the compiled binary.  Add it to PATH in case
	// users wish to invoke the binary manually.
	bl, err := ctx.Layer("bin", gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	bl.LaunchEnvironment.Prepend("PATH", string(os.PathListSeparator), bl.Path)
	outBin := filepath.Join(bl.Path, "server")

	// Build the server first
	buildable, ok := os.LookupEnv(env.Buildable)
	if !ok {
		buildable = "bin/server.dart"
	}
	bld := []string{"dart", "compile", "exe", buildable, "-o", outBin}
	if _, err := ctx.Exec(bld, gcp.WithUserAttribution, gcp.WithWorkDir(server)); err != nil {
		return err
	}
	ctx.AddWebProcess([]string{"/bin/bash", "-c", outBin})

	// Build the webapp
	// "--output", filepath.Join(bl.Path, "static")
	bld = []string{"flutter", "build", "web"} // /workspace/<app>/build/web
	if _, err := ctx.Exec(bld, gcp.WithUserAttribution, gcp.WithWorkDir(static)); err != nil {
		return err
	}

	if rootPubspec.Buildpack.Postbuild != nil {
		if _, err := ctx.Exec([]string{"sh", "-c", *rootPubspec.Buildpack.Postbuild}, gcp.WithUserAttribution, gcp.WithWorkDir(bl.Path)); err != nil {
			return err
		}
	}

	return nil
}
