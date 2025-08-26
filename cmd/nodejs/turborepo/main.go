// Copyright 2025 Google LLC
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

// Implements nodejs/turborepo buildpack.
// The nodejs/turborepo buildpack analyzes and configures a build for monorepos using Turborepo.
package main

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

var (
	versionKey = "version"
	// The Turbo buildpack build function sets the following environment variables to configure the build
	// behavior of subsequent buildpacks.
	monorepoProject   = "MONOREPO_PROJECT"    // The name of the target application in a turbo monorepo.
	monorepoCommand   = "MONOREPO_COMMAND"    // The CLI command utility ("turbo").
	monorepoBuildArgs = "MONOREPO_BUILD_ARGS" // The build arguments to pass to the turbo command.
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	turboJSONExists, err := ctx.FileExists("turbo.json")
	if err != nil {
		return nil, err
	}
	if !turboJSONExists {
		return gcp.OptOutFileNotFound("turbo.json"), nil
	}
	return gcp.OptInFileFound("turbo.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	appDir := util.ApplicationDirectory(ctx)

	turboJSON, err := nodejs.ReadTurboJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if turboJSON == nil {
		return gcp.UserErrorf("turbo.json file does not exist")
	}

	appPackageJSON, err := nodejs.ReadPackageJSONIfExists(appDir)
	if err != nil {
		return err
	}

	var appName string
	if appPackageJSON != nil {
		appName = appPackageJSON.Name
	}
	// Target application is ambiguous, so we fail the build.
	if appName == "" {
		return gcp.UserErrorf("target application in Turbo monorepo is ambiguous. Please specify the application directory path during onboarding.")
	}

	buildArgs := []string{fmt.Sprintf("--filter=%s", appName), "--env-mode=loose"}

	turbol, err := ctx.Layer("turbo", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating turbo layer: %w", err)
	}
	// Set environment variables that will configure the build to use Turborepo.
	turbol.BuildEnvironment.Override(monorepoProject, appName)
	turbol.BuildEnvironment.Override(monorepoCommand, "turbo")
	turbol.BuildEnvironment.Override(monorepoBuildArgs, strings.Join(buildArgs, ","))

	// add turbo as the monorepo name to the builder metadata
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.MonorepoName, "turbo")

	return nil
}
