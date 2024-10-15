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

// Implements nodejs/firebasenx buildpack.
// The nodejs/firebasenx buildpack analyzes and configures a build for Nx monorepos.
package main

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

var (
	versionKey = "version"
	// The Nx buildpack build function sets the following environment variables to configure the build
	// behavior of subsequent buildpacks.
	monorepoProject   = "MONOREPO_PROJECT"    // The name of a project in a Nx monorepo.
	monorepoCommand   = "MONOREPO_COMMAND"    // The CLI command utility ("nx").
	monorepoBuildArgs = "MONOREPO_BUILD_ARGS" // The builder plugin used by the build target executor.
	nxNoCloud         = "NX_NO_CLOUD"         // Whether to disable Nx Cloud remote caching.
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsFAH() {
		return gcp.OptOut("not a firebase apphosting application"), nil
	}
	nxJSONExists, err := ctx.FileExists("nx.json")
	if err != nil {
		return nil, err
	}
	if !nxJSONExists {
		return gcp.OptOutFileNotFound("nx.json"), nil
	}
	return gcp.OptInFileFound("nx.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	appDir := util.ApplicationDirectory(ctx)

	nxJSON, err := nodejs.ReadNxJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if nxJSON == nil {
		return gcp.UserErrorf("nx.json file does not exist")
	}
	if nxJSON.NxCloudAccessToken != "" {
		ctx.Warnf("Nx Cloud is currently not supported. Ignoring Nx Cloud Access Token")
	}

	projectJSON, err := nodejs.ReadNxProjectJSONIfExists(appDir)
	if err != nil {
		return err
	}

	projectName := nxJSON.DefaultProject
	if projectJSON != nil {
		projectName = projectJSON.Name
	}
	// Project target is ambiguous, so we fail the build.
	if projectName == "" {
		return gcp.UserErrorf("target application in Nx monorepo is ambiguous. Please specify the application directory path during onboarding or a default project in nx.json")
	}

	buildArgs := []string{fmt.Sprintf("--project=%s", projectName)}

	nxl, err := ctx.Layer("nx", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating nx layer: %w", err)
	}
	// Set environment variables that will configure the build to use Nx.
	nxl.BuildEnvironment.Override(monorepoProject, projectName)
	nxl.BuildEnvironment.Override(monorepoCommand, "nx")
	nxl.BuildEnvironment.Override(monorepoBuildArgs, strings.Join(buildArgs, ","))
	// If an Nx Cloud access token is provided in nx.json, the Nx build script will attempt to read
	// from the user's cloud cache. This feature is currently disabled so that users don't rely on Nx
	// remote caching until we can fully investigate how to support making external network requests
	// to the Nx API.
	nxl.BuildEnvironment.Override(nxNoCloud, "true")

	return nil
}
