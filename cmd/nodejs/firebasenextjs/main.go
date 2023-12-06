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
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/Masterminds/semver"
)

var (
	// minNextVersion is the lowest version of nextjs supported by the firebasenextjs buildpack.
	minNextVersion = semver.MustParse("13.0.0")
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// TODO (b/311402770)
	// In monorepo scenarios, we'll probably need to support environment variable that can be used to
	// know where the application directory is located within the repository.
	// TODO (b/313959098)
	// Verify nextjs version
	nextConfigExists, err := ctx.FileExists("next.config.js")
	if err != nil {
		return nil, err
	}
	if nextConfigExists {
		return gcp.OptInFileFound("next.config.js"), nil
	}

	nextConfigModuleExists, err := ctx.FileExists("next.config.mjs")
	if err != nil {
		return nil, err
	}
	if nextConfigModuleExists {
		return gcp.OptInFileFound("next.config.mjs"), nil
	}
	return gcp.OptOut("nextjs config not found"), nil
}

func buildFn(ctx *gcp.Context) error {
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	version, err := semver.NewVersion(pjs.Dependencies["next"])
	if err != nil {
		return err
	}

	if version.LessThan(minNextVersion) {
		return gcp.UserErrorf("running unsupported nextjs version, minimum %s required but %s is being run", minNextVersion.String(), version.String())
	}

	buildScript, exists := pjs.Scripts["build"]

	if exists && buildScript == "next build" {
		njsl, err := ctx.Layer("npm_modules", gcp.BuildLayer, gcp.CacheLayer)
		if err != nil {
			return err
		}
		err = nodejs.InstallNextJsBuildAdaptor(ctx, njsl)
		if err != nil {
			return err
		}
		// This env var indicates to the package manager buildpack that a different command needs to be run
		nodejs.OverrideNextjsBuildScript(njsl)
	} else if exists && buildScript != "apphosting-adapter-nextjs-build" {
		ctx.Warnf("*** You are using a custom build command (your build command is NOT 'next build'), we will accept it as is but some features will not be enabled ***")
	}
	return err
}
