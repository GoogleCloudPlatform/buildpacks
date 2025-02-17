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

// Implements python/pip buildpack.
// The pip buildpack installs dependencies using pip.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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

func buildFn(ctx *gcp.Context) error {
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

	l, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	if err := python.InstallRequirements(ctx, l, reqs...); err != nil {
		return fmt.Errorf("installing dependencies: %w", err)
	}

	ctx.Logf("Checking for incompatible dependencies.")
	result, err := ctx.Exec([]string{"python3", "-m", "pip", "check"}, gcp.WithUserAttribution)
	if result == nil {
		return fmt.Errorf("pip check: %w", err)
	}
	if result.ExitCode == 0 {
		return nil
	}
	pyVer, err := python.Version(ctx)
	if err != nil {
		return err
	}
	// HACK: For backwards compatibility on App Engine and Cloud Functions Python 3.7 only report a warning.
	if strings.HasPrefix(pyVer, "Python 3.7") {
		ctx.Warnf("Found incompatible dependencies: %q", result.Stdout)
		return nil
	}
	return gcp.UserErrorf("found incompatible dependencies: %q", result.Stdout)

}
