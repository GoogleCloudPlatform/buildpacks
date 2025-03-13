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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/Masterminds/semver"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	// frameworkVersion is the version of next that the application is using
	frameworkVersion = "FRAMEWORK_VERSION"
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

	if !env.IsFAH() {
		return gcp.OptOut("not a firebase apphosting application"), nil
	}
	nodeDeps, err := nodejs.ReadNodeDependencies(ctx, appDir)
	if err != nil {
		return nil, err
	}
	apphostingSchema, err := apphostingschema.ReadAndValidateFromFile(nodejs.ApphostingPreprocessedPathForPack)
	if err != nil {
		return nil, err
	}
	if nodejs.HasApphostingPackageOrYamlBuild(nodeDeps.PackageJSON, apphostingSchema) {
		return gcp.OptOut("apphosting build script found"), nil
	}

	supportedNextConfigFiles := []string{"next.config.js", "next.config.mjs", "next.config.ts"}

	for _, configFile := range supportedNextConfigFiles {
		exists, err := ctx.FileExists(appDir, configFile)
		if err != nil {
			return nil, err
		}
		if exists {
			return gcp.OptInFileFound(configFile), nil
		}
	}

	version, err := nodejs.Version(nodeDeps, "next")
	if err != nil {
		ctx.Warnf("Error parsing version from lock file, defaulting to package.json version")
		version = nodeDeps.PackageJSON.Dependencies["next"]
	}
	if version != "" {
		return gcp.OptIn("nextjs dependency found"), nil
	}

	return gcp.OptOut("nextjs config or dependency not found"), nil
}

func buildFn(ctx *gcp.Context) error {
	appDir := util.ApplicationDirectory(ctx)

	nodeDeps, err := nodejs.ReadNodeDependencies(ctx, appDir)
	if err != nil {
		return err
	}
	if nodeDeps.LockfilePath == "" {
		return gcp.UserErrorf("%w", faherror.MissingLockFileError(appDir))
	}

	version, err := nodejs.Version(nodeDeps, "next")
	if err != nil {
		ctx.Warnf("Error parsing version from lock file, defaulting to package.json version")
		version = nodeDeps.PackageJSON.Dependencies["next"]
	}
	err = validateVersion(ctx, version)
	if err != nil {
		return err
	}

	// TODO(b/357644160) We we should consider adding a validation step to double check that the adapter version works for the framework version.
	if version, exists := nodeDeps.PackageJSON.Dependencies["@apphosting/adapter-nextjs"]; exists {
		ctx.Logf("*** You already have @apphosting/adapter-nextjs@%s listed as a dependency, skipping installation ***", version)
		ctx.Logf("*** Your package.json build command will be run as is, please make sure it is set to apphosting-adapter-nextjs-build if you intend to build your app using the adapter ***")
		return nil
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

	// pass nextjs version as environment variable that will configure the build for version matching
	njsl.BuildEnvironment.Override(frameworkVersion, version)

	// This env var indicates to the package manager buildpack that a different command needs to be run
	nodejs.OverrideNextjsBuildScript(njsl)

	return nil
}

func validateVersion(ctx *gcp.Context, depVersion string) error {
	version, err := semver.NewVersion(depVersion)
	// This should only happen in the case of an unexpected lockfile format, i.e. If there is a breaking update to a lock file schema
	if err != nil {
		ctx.Warnf("Unrecognized version of next: %s", depVersion)
		ctx.Warnf("Consider updating your next dependencies to >=%s", minNextVersion.String())
		return nil
	}
	if version.LessThan(minNextVersion) {
		ctx.Warnf("Unsupported version of next: %s", depVersion)
		ctx.Warnf("Update the next dependencies to >=%s", minNextVersion.String())
		return gcp.UserErrorf("%w", faherror.UnsupportedFrameworkVersionError("next", depVersion))
	}
	return nil
}
