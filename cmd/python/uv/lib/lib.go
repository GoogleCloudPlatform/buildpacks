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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName = "uv-dependencies"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	isUVPyproject, message, err := python.IsUVPyproject(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}
	if isUVPyproject {
		return gcp.OptIn(message), nil
	}

	plan := libcnb.BuildPlan{Requires: python.RequirementsRequires}
	// If a requirement.txt file exists, the buildpack needs to provide the Requirements dependency.
	// If the dependency is not provided by any buildpacks, lifecycle will exclude the uv
	// buildpack from the build.
	requirementsExists, err := ctx.FileExists("requirements.txt")
	if err != nil {
		return nil, err
	}
	if requirementsExists {
		plan.Provides = python.RequirementsProvides
	}

	isUVRequirements, message, err := python.IsUVRequirements(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}
	if isUVRequirements {
		return gcp.OptIn(message, gcp.WithBuildPlans(plan)), nil
	}

	return gcp.OptOut(message), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.UVUsageCounterID).Increment(1)
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, buildermetadata.MetadataValue("uv"))

	if err := python.InstallUV(ctx); err != nil {
		return fmt.Errorf("installing uv: %w", err)
	}

	l, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	isUVPyproject, _, err := python.IsUVPyproject(ctx)
	if err != nil {
		return err
	}

	if isUVPyproject {
		buildermetadata.GlobalBuilderMetadata().SetValue(
			buildermetadata.ConfigFile,
			buildermetadata.MetadataValue("pyproject.toml"),
		)
		if _, err = python.UVInstallDependenciesAndConfigureEnv(ctx, l); err != nil {
			return fmt.Errorf("installing dependencies with uv: %w", err)
		}
	} else {
		// This is the requirements.txt path
		buildermetadata.GlobalBuilderMetadata().SetValue(
			buildermetadata.ConfigFile,
			buildermetadata.MetadataValue("requirements.txt"),
		)
		reqs := filepath.SplitList(strings.Trim(os.Getenv(python.RequirementsFilesEnv), string(os.PathListSeparator)))
		ctx.Debugf("Found requirements.txt files provided by other buildpacks: %s", reqs)

		// The workspace requirements.txt file should be installed last.
		requirementsExists, err := ctx.FileExists("requirements.txt")
		if err != nil {
			return err
		}
		if requirementsExists {
			reqs = append(reqs, "requirements.txt")
		}

		ctx.Logf("Found requirements.txt, installing with `uv pip install`.")
		if _, err = python.UVInstallRequirements(ctx, l, reqs...); err != nil {
			return gcp.UserErrorf("installing requirements.txt with uv: %w", err)
		}
	}

	return nil
}
