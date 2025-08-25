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

// Implements python/poetry buildpack.
// The poetry buildpack installs dependencies using poetry.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsAlphaSupported() {
		return gcp.OptOut("Python Poetry Buildpack is only supported in ALPHA release tracks."), nil
	}

	isPoetry, message, err := python.IsPoetryProject(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}

	if isPoetry {
		return gcp.OptIn(message), nil
	}
	return gcp.OptOut(message), nil
}

func buildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PoetryUsageCounterID).Increment(1)
	// Install Poetry.
	if err := python.InstallPoetry(ctx); err != nil {
		return fmt.Errorf("installing poetry: %w", err)
	}

	// Ensure poetry.lock exists or generate it.
	if err := python.EnsurePoetryLockfile(ctx); err != nil {
		return fmt.Errorf("ensuring poetry.lock: %w", err)
	}

	// Install dependencies and configure the environment.
	if err := python.PoetryInstallDependenciesAndConfigureEnv(ctx); err != nil {
		return fmt.Errorf("installing dependencies and configuring env: %w", err)
	}

	// Check for incompatible dependencies.
	if _, err := ctx.Exec([]string{"poetry", "check"}, gcp.WithUserAttribution); err != nil {
		ctx.Logf("Warning: 'poetry check' returned an error, which might just be a deprecation warning: %v", err)
	} else {
		ctx.Debugf("No incompatible dependencies found.")
	}

	return nil
}
