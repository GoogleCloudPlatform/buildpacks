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

// Implements python/webserver buildpack.
// The webserver buildpack installs gunicorn if a custom entrypoint is not specified.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

const (
	layerName = "gunicorn"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

var (
	gunicorn = "gunicorn"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if os.Getenv(env.Entrypoint) != "" {
		return gcp.OptOut("custom entrypoint present"), nil
	}
	requirementsExists, err := ctx.FileExists("requirements.txt")
	if err != nil {
		return nil, err
	}
	if requirementsExists {
		present, err := python.RequirementsPackagePresent(ctx, gunicorn)
		if err != nil {
			return nil, fmt.Errorf("error detecting gunicorn: %w", err)
		}
		if present {
			return gcp.OptOut("gunicorn present in requirements.txt"), nil
		}
		return gcp.OptIn("gunicorn missing from requirements.txt", gcp.WithBuildPlans(python.RequirementsProvidesPlan)), nil
	}
	return gcp.OptIn("requirements.txt with gunicorn not found", gcp.WithBuildPlans(python.RequirementsProvidesPlan)), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer(layerName, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	// The pip install is performed by the pip buildpack; see python.InstallRequirements.
	ctx.Debugf("Adding webserver requirements.txt to the list of requirements files to install.")
	r := filepath.Join(ctx.BuildpackRoot(), "requirements.txt")
	l.BuildEnvironment.Append(python.RequirementsFilesEnv, string(os.PathListSeparator), r)
	return nil
}
