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

// Implements nodejs/firebasenextjs buildpack.
// The nodejs/firebasenextjs buildpack does some prep work for nextjs and overwrites the build script.
package main

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/Masterminds/semver"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// minNextVersion is the lowest version of nextjs supported by the firebasenextjs buildpack.
	minNextVersion = semver.MustParse("13.0.0")
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	appDir := util.ApplicationDirectory(ctx)
	// TODO (b/313959098)
	// Verify nextjs version
	nextConfigExists, err := ctx.FileExists(appDir, "next.config.js")
	if err != nil {
		return nil, err
	}
	if nextConfigExists {
		return gcp.OptInFileFound("next.config.js"), nil
	}

	nextConfigModuleExists, err := ctx.FileExists(appDir, "next.config.mjs")
	if err != nil {
		return nil, err
	}
	if nextConfigModuleExists {
		return gcp.OptInFileFound("next.config.mjs"), nil
	}
	return gcp.OptOut("nextjs config not found"), nil
}

func buildFn(ctx *gcp.Context) error {
	appDir := util.ApplicationDirectory(ctx)

	nodeDeps, err := nodejs.ReadNodeDependencies(ctx, appDir)
	if err != nil {
		return err
	}
	version, err := nodejs.Version(nodeDeps, "next")
	if err != nil {
		return err
	}
	err = validateVersion(ctx, version)
	if err != nil {
		return err
	}

	buildScript, exists := nodeDeps.PackageJSON.Scripts["build"]
	if exists && buildScript != "next build" && buildScript != "apphosting-adapter-nextjs-build" {
		ctx.Warnf("*** You are using a custom build command (your build command is NOT 'next build'), we will accept it as is but will error if output structure is not as expected ***")
	}

	njsl, err := ctx.Layer("npm_modules", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return err
	}
	err = nodejs.InstallNextJsBuildAdaptor(ctx, njsl, version)
	if err != nil {
		return err
	}
	// This env var indicates to the package manager buildpack that a different command needs to be run
	nodejs.OverrideNextjsBuildScript(njsl)

	return err
}

func validateVersion(ctx *gcp.Context, depVersion string) error {
	version, err := semver.NewVersion(depVersion)
	if err != nil {
		return gcp.InternalErrorf("parsing next version: %v", err)
	}
	if version.LessThan(minNextVersion) {
		ctx.Warnf("Unsupported version of next: %s", depVersion)
		ctx.Warnf("Update the next dependencies to >=%s", minNextVersion.String())
		return gcp.UserErrorf("unsupported version of next %s", depVersion)
	}
	return nil
}
