// Copyright 2020 Google LLC
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
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !python.IsPyprojectEnabled(ctx) {
		return gcp.OptOut("Python UV Buildpack is not supported in the current release track."), nil
	}

	isUV, message, err := python.IsUVProject(ctx)
	if err != nil {
		return gcp.OptOut(message), err
	}

	if isUV {
		return gcp.OptIn(message), nil
	}
	return gcp.OptOut(message), nil
}

func buildFn(ctx *gcp.Context) error {
	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.UVUsageCounterID).Increment(1)
	if err := python.InstallUV(ctx); err != nil {
		return fmt.Errorf("installing uv: %w", err)
	}

	if err := python.EnsureUVLockfile(ctx); err != nil {
		return fmt.Errorf("ensuring uv.lock file: %w", err)
	}

	if err := python.UVInstallDependenciesAndConfigureEnv(ctx); err != nil {
		return fmt.Errorf("installing dependencies with uv: %w", err)
	}

	return nil
}
