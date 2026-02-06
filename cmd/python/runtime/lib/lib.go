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

// Implements python/runtime buildpack.
// The runtime buildpack installs the Python runtime.
package lib

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/flex"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	pythonLayer = "python"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if flex.NeedsSupervisorPackage(ctx) {
		return gcp.OptIn("supervisor package is required"), nil
	}

	if result := runtime.CheckOverride("python"); result != nil {
		return result, nil
	}
	atLeastOne, err := ctx.HasAtLeastOneOutsideDependencyDirectories("*.py")
	if err != nil {
		return nil, fmt.Errorf("finding *.py files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .py files found"), nil
	}
	return gcp.OptIn("found .py files"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	// We don't cache the python runtime because the python/link-runtime buildpack may clobber
	// everything in the layer directory anyway.
	layer, err := ctx.Layer(pythonLayer, gcp.BuildLayer, gcp.LaunchLayer)
	ctx.Logf("layers path: %s", layer.Path)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", pythonLayer, err)
	}
	ver, err := python.RuntimeVersion(ctx, ctx.ApplicationRoot())
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	if _, err := runtime.InstallTarballIfNotCached(ctx, runtime.Python, ver, layer); err != nil {
		return err
	}

	if err := python.PatchSysconfig(ctx, layer); err != nil {
		return err
	}

	// Set the PYTHONHOME for flex apps because of uwsgi
	if env.IsFlex() {
		layer.LaunchEnvironment.Default("PYTHONHOME", layer.Path)
	}
	// Force stdout/stderr streams to be unbuffered so that log messages appear immediately in the logs.
	layer.LaunchEnvironment.Default("PYTHONUNBUFFERED", "TRUE")
	return nil
}
