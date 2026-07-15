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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devsync"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
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

	if enabled, _ := env.IsDevSyncUseRunitUniversalMaker(); !enabled {
		return gcp.OptOut("X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER is not set to true"), nil
	}

	return gcp.OptIn("GOOGLE_DEVSYNC and X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER are true"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	layer, err := ctx.Layer("devsync", gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating devsync layer: %w", err)
	}

	// Guarantee GOOGLE_DEVSYNC=1 is preserved in the container's runtime environment
	// so that universal_maker inherits it during live incremental rebuilds.
	layer.LaunchEnvironment.Default(env.DevSync, "1")

	if err := installMaker(ctx, layer); err != nil {
		return fmt.Errorf("installing universal_maker: %w", err)
	}

	if err := configureRunitServiceTree(ctx, layer); err != nil {
		return fmt.Errorf("configuring runit service tree: %w", err)
	}

	return nil
}

func installMaker(ctx *gcp.Context, layer *libcnb.Layer) error {
	binDir := filepath.Join(layer.Path, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	// TODO(b/521268420): Integrate with tooling.textproto versioning system.
	version := "1.0.1"
	binaryPath := filepath.Join(binDir, "universal_maker")

	if err := fetch.ARGenericBinary(ctx, "universal_maker", version, binaryPath); err != nil {
		return err
	}

	return nil
}

func configureRunitServiceTree(ctx *gcp.Context, layer *libcnb.Layer) error {
	serviceDir := filepath.Join(layer.Path, "service")
	if err := fileutil.MaybeCopyPathContents(serviceDir, filepath.Join(ctx.BuildpackRoot(), "service"), fileutil.AllPaths); err != nil {
		return fmt.Errorf("copying service tree: %w", err)
	}

	webCmd := os.Getenv(env.DevSyncInitEntrypoint)
	if webCmd == "" {
		webCmd = "echo 'No web process found'"
	}

	if err := devsync.UpdateAppRunScript(serviceDir, webCmd, nil); err != nil {
		return err
	}

	if err := os.Chmod(filepath.Join(serviceDir, "watcher", "run"), 0755); err != nil {
		return fmt.Errorf("chmoding watcher/run: %w", err)
	}
	if err := os.Chmod(filepath.Join(serviceDir, "app", "control", "t"), 0755); err != nil {
		return fmt.Errorf("chmoding app/control/t: %w", err)
	}

	ctx.Logf("Setting web process: runsvdir %s", serviceDir)
	ctx.AddWebProcess([]string{"runsvdir", serviceDir})

	return nil
}
