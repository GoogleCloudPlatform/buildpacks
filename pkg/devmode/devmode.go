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

// Package devmode contains helpers to configure Development Mode.
package devmode

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	watchexecLayer   = "watchexec"
	watchexecVersion = "1.12.0"
	watchexecURL     = "https://github.com/watchexec/watchexec/releases/download/%[1]s/watchexec-%[1]s-x86_64-unknown-linux-gnu.tar.xz"
	scriptsLayer     = "devmode_scripts"
	buildAndRun      = "build_and_run.sh"

	// WatchAndRun is the name of the script that watches source files and runs the
	// build_and_run.sh script when those files change.
	WatchAndRun = "watch_and_run.sh"
)

// SyncRule represents a sync rule.
type SyncRule struct {
	// Src is a glob, and assumed to be a path relative to the user's workspace.
	Src string `toml:"src"`

	// Dest is the destination root folder where changed files are copied.
	// Relative directory structure is preserved while copying.
	Dest string `toml:"dest"`
}

// Enabled indicates that the builder is running in Development mode.
func Enabled(ctx *gcp.Context) bool {
	devMode, present := os.LookupEnv(env.DevMode)
	if !present {
		return false
	}

	enabled, err := strconv.ParseBool(devMode)
	if err != nil {
		ctx.Warnf("%s env var must be parseable to a bool: %q", env.DevMode, devMode)
		return false
	}

	return enabled
}

// metadata represents metadata stored for a devmode layer.
type metadata struct {
	WatchexecVersion string `toml:"version"`
}

// Config describes the dev mode for a given language.
type Config struct {
	BuildCmd []string
	RunCmd   []string
	// Ext lists the file extensions that trigger a restart.
	Ext []string
}

// AddFileWatcherProcess installs and configures a file watcher as the entrypoint.
func AddFileWatcherProcess(ctx *gcp.Context, cfg Config) {
	installFileWatcher(ctx)
	writeBuildAndRunScript(ctx, ctx.Layer(scriptsLayer), cfg)
	// Override the web process.
	ctx.AddWebProcess([]string{WatchAndRun})
}

// AddSyncMetadata adds sync metadata to the final image.
func AddSyncMetadata(ctx *gcp.Context, syncRulesFn func(string) []SyncRule) {
	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Metadata: buildpackplan.Metadata{
			"devmode.sync": syncRulesFn(ctx.ApplicationRoot()),
		},
	})
}

// writeBuildAndRunScript writes the contents of a file that builds code and then runs the resulting program
func writeBuildAndRunScript(ctx *gcp.Context, sl *layers.Layer, cfg Config) {

	binDir := filepath.Join(sl.Root, "bin")
	ctx.MkdirAll(binDir, 0755)

	var cmd []string
	if cfg.BuildCmd != nil {
		cmd = append(cmd, strings.Join(cfg.BuildCmd, " "))
	}
	if cfg.RunCmd != nil {
		cmd = append(cmd, strings.Join(cfg.RunCmd, " "))
	}

	c := fmt.Sprintf("#!/bin/sh\n%s", strings.Join(cmd, " && "))
	br := filepath.Join(binDir, buildAndRun)
	ctx.WriteFile(br, []byte(c), os.FileMode(0755))

	c = fmt.Sprintf("#!/bin/sh\nwatchexec -r -e %s %s", strings.Join(cfg.Ext, ","), br)
	wr := filepath.Join(binDir, WatchAndRun)
	ctx.WriteFile(wr, []byte(c), os.FileMode(0755))

	ctx.WriteMetadata(sl, nil, layers.Launch)
}

// installFileWatcher installs the `watchexec` file watcher.
func installFileWatcher(ctx *gcp.Context) {
	wxl := ctx.Layer(watchexecLayer)

	// Check metadata layer to see if correct version of watchexec is already installed.
	var meta metadata
	ctx.ReadMetadata(wxl, &meta)
	if meta.WatchexecVersion == watchexecVersion {
		ctx.CacheHit(watchexecLayer)
	} else {
		ctx.CacheMiss(watchexecLayer)
		// Clear layer data to avoid files from multiple versions of watchexec.
		ctx.ClearLayer(wxl)

		binDir := filepath.Join(wxl.Root, "bin")
		ctx.MkdirAll(binDir, 0755)

		// Download and install watchexec in layer.
		ctx.Logf("Installing watchexec v%s", watchexecVersion)
		archiveURL := fmt.Sprintf(watchexecURL, watchexecVersion)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xJ --directory %s --strip-components=1 --wildcards \"*watchexec\"", archiveURL, binDir)
		ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

		meta.WatchexecVersion = watchexecVersion
	}

	// Write the layer information.
	ctx.WriteMetadata(wxl, meta, layers.Cache, layers.Launch)
}
