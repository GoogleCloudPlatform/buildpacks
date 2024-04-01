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
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

var (
	versionKey           = "version"
	firebaseAppDirectory = "FIREBASE_APP_DIRECTORY"
	// The Nx buildpack build function sets the following environment variables to configure the build
	// behavior of subsequent buildpacks.
	monorepoProject = "MONOREPO_PROJECT" // The name of a project in a Nx monorepo.
	monorepoBuilder = "MONOREPO_BUILDER" // The builder plugin used by the build target executor.
	monorepoCommand = "MONOREPO_COMMAND" // The CLI command utility ("nx").
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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
	var appDir string
	if appDirEnv, exists := os.LookupEnv(firebaseAppDirectory); exists {
		appDir = filepath.Join(ctx.ApplicationRoot(), appDirEnv)
	} else {
		return gcp.UserErrorf("%s not specified", firebaseAppDirectory)
	}

	projectJSON, err := nodejs.ReadNxProjectJSONIfExists(appDir)
	if err != nil {
		return err
	}
	if projectJSON == nil {
		return gcp.UserErrorf("%s should include a project.json file.", appDir)
	}
	projectName := projectJSON.Name
	projectBuilder := projectJSON.Targets.Build.Executor

	nxl, err := ctx.Layer("nx", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating nx layer: %w", err)
	}
	// Set environment variables that will configure the build to use Nx.
	nxl.BuildEnvironment.Override(monorepoProject, projectName)
	nxl.BuildEnvironment.Override(monorepoBuilder, projectBuilder)
	nxl.BuildEnvironment.Override(monorepoCommand, "nx")

	return nil
}
