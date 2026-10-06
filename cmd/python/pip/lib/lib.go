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

// Implements python/pip buildpack.
// The pip buildpack installs dependencies using pip.
package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName = "pip"
)

// metadata represents metadata stored for a dependencies layer.
type metadata struct {
	PythonVersion   string `toml:"python_version"`
	DependencyHash  string `toml:"dependency_hash"`
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if python.IsPipPyproject(ctx) {
		return gcp.OptIn(fmt.Sprintf("found pyproject.toml, using pip because %s is set to 'pip' ", env.PythonPackageManager)), nil
	}
	plan := libcnb.BuildPlan{Requires: python.RequirementsRequires}
	// If a requirement.txt file exists, the buildpack needs to provide the Requirements dependency.
	// If the dependency is not provided by any buildpacks, lifecycle will exclude the pip
	// buildpack from the build.
	requirementsExists, err := ctx.FileExists("requirements.txt")
	if err != nil {
		return nil, err
	}
	if requirementsExists {
		plan.Provides = python.RequirementsProvides
	}
	return gcp.OptInAlways(gcp.WithBuildPlans(plan)), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PIPUsageCounterID).Increment(1)
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.PackageManager, buildermetadata.MetadataValue("pip"))

	l, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	// Check if this build is for a pyproject.toml file.
	if python.IsPipPyproject(ctx) {
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.ConfigFile, buildermetadata.MetadataValue("pyproject.toml"))
		if err := python.PipInstallPyproject(ctx, l); err != nil {
			return gcp.UserErrorf("installing dependencies from pyproject.toml: %w", err)
		}
	} else {
		// Fallback to the requirements.txt logic.
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.ConfigFile, buildermetadata.MetadataValue("requirements.txt"))
		// Remove leading and trailing : because otherwise SplitList will add empty strings.
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

		if err := python.PIPInstallRequirements(ctx, l, reqs...); err != nil {
			return fmt.Errorf("installing dependencies from requirements.txt and validating them: %w", err)
		}
	}

	return nil
}
