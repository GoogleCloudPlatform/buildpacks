// Copyright 2026 Google LLC
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

// Package lib implements utils/devsync buildpack.
// The devsync buildpack installs the universal_maker binary into a launch layer.
package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

// SkipDevsyncCapability is the capability key used by Maker to skip this buildpack.
const SkipDevsyncCapability = "devsync.Skip"

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.Capability(SkipDevsyncCapability) != nil {
		return gcp.OptOut("Running as maker, skipping devsync buildpack"), nil
	}

	if active, _ := env.IsDevSync(); !active {
		return gcp.OptOut("not a devsync build"), nil
	}

	return gcp.OptIn("GOOGLE_DEVSYNC is true"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	layer, err := ctx.Layer("devsync", gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating devsync layer: %w", err)
	}

	if err := installMaker(ctx, layer); err != nil {
		return fmt.Errorf("installing universal_maker: %w", err)
	}

	return nil
}

func installMaker(ctx *gcp.Context, layer *libcnb.Layer) error {
	binDir := filepath.Join(layer.Path, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	// TODO(b/521268420): Integrate with tooling.textproto versioning system.
	version := "1.0.0"
	binaryPath := filepath.Join(binDir, "universal_maker")

	if err := fetch.ARGenericBinary(ctx, "universal_maker", version, binaryPath); err != nil {
		return err
	}

	return nil
}
