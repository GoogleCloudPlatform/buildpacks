// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Implements python/uv buildpack.
// The uv buildpack installs dependencies using uv.
package lib

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	isUVPyproject, message, err := python.IsUVPyproject(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}
	if isUVPyproject {
		if !python.IsPyprojectEnabled(ctx) {
			return gcp.OptOut("Python UV Buildpack for pyproject.toml is not supported in the current release track."), nil
		}
		return gcp.OptIn(message), nil
	}

	isUVRequirements, message, err := python.IsUVRequirements(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}
	if isUVRequirements {
		if !python.IsUVRequirementsEnabled(ctx) {
			return gcp.OptOut("Python UV Buildpack for requirements.txt is not supported in the current release track."), nil
		}
		return gcp.OptIn(message), nil
	}

	return gcp.OptOut(message), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.UVUsageCounterID).Increment(1)
	if err := python.InstallUV(ctx); err != nil {
		return fmt.Errorf("installing uv: %w", err)
	}

	isUVRequirements, _, err := python.IsUVRequirements(ctx)
	if err != nil {
		return fmt.Errorf("checking for uv requirements: %w", err)
	}

	if isUVRequirements {
		ctx.Logf("Found requirements.txt, installing with `uv pip install`.")
		if err := python.UVInstallRequirements(ctx); err != nil {
			return gcp.UserErrorf("installing requirements.txt with uv: %w", err)
		}
		return nil
	}

	if err := python.EnsureUVLockfile(ctx); err != nil {
		return fmt.Errorf("ensuring uv.lock file: %w", err)
	}

	if err := python.UVInstallDependenciesAndConfigureEnv(ctx); err != nil {
		return fmt.Errorf("installing dependencies with uv: %w", err)
	}

	return nil
}
