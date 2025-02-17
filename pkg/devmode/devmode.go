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
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

const (
	watchexecLayer   = "watchexec"
	watchexecVersion = "1.12.0"
	watchexecURL     = "https://github.com/watchexec/watchexec/releases/download/%[1]s/watchexec-%[1]s-x86_64-unknown-linux-gnu.tar.xz"
	scriptsLayer     = "devmode_scripts"
	buildAndRun      = "build_and_run.sh"
	versionKey       = "version"

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
	enabled, err := env.IsDevMode()
	if err != nil {
		ctx.Warnf("Dev mode not enabled: %v", err)
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
func AddFileWatcherProcess(ctx *gcp.Context, cfg Config) error {
	installFileWatcher(ctx)
	sl, err := ctx.Layer(scriptsLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", scriptsLayer, err)
	}
	writeBuildAndRunScript(ctx, sl, cfg)
	// Override the web process.
	ctx.AddWebProcess([]string{WatchAndRun})
	return nil
}

// writeBuildAndRunScript writes the contents of a file that builds code and then runs the resulting program
func writeBuildAndRunScript(ctx *gcp.Context, sl *libcnb.Layer, cfg Config) error {
	sl.Launch = true
	binDir := filepath.Join(sl.Path, "bin")
	if err := ctx.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	var cmd []string
	if cfg.BuildCmd != nil {
		cmd = append(cmd, strings.Join(cfg.BuildCmd, " "))
	}
	if cfg.RunCmd != nil {
		cmd = append(cmd, strings.Join(cfg.RunCmd, " "))
	}

	c := fmt.Sprintf("#!/bin/sh\n%s", strings.Join(cmd, " && "))
	br := filepath.Join(binDir, buildAndRun)
	if err := ctx.WriteFile(br, []byte(c), os.FileMode(0755)); err != nil {
		return err
	}

	c = fmt.Sprintf("#!/bin/sh\nwatchexec -r -e %s %s", strings.Join(cfg.Ext, ","), br)
	wr := filepath.Join(binDir, WatchAndRun)
	if err := ctx.WriteFile(wr, []byte(c), os.FileMode(0755)); err != nil {
		return err
	}
	return nil
}

// installFileWatcher installs the `watchexec` file watcher.
func installFileWatcher(ctx *gcp.Context) error {
	wxl, err := ctx.Layer(watchexecLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", watchexecLayer, err)
	}

	// Check metadata layer to see if correct version of watchexec is already installed.
	metaWatchexecVersion := ctx.GetMetadata(wxl, versionKey)
	if metaWatchexecVersion == watchexecVersion {
		ctx.CacheHit(watchexecLayer)
	} else {
		ctx.CacheMiss(watchexecLayer)
		// Clear layer data to avoid files from multiple versions of watchexec.
		if err := ctx.ClearLayer(wxl); err != nil {
			return fmt.Errorf("clearing layer %q: %w", wxl.Name, err)
		}

		binDir := filepath.Join(wxl.Path, "bin")
		if err := ctx.MkdirAll(binDir, 0755); err != nil {
			return err
		}

		// Download and install watchexec in layer.
		ctx.Logf("Installing watchexec v%s", watchexecVersion)
		archiveURL := fmt.Sprintf(watchexecURL, watchexecVersion)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xJ --directory %s --strip-components=1 --wildcards \"*watchexec\"", archiveURL, binDir)
		if _, err := ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution); err != nil {
			return err
		}
		ctx.SetMetadata(wxl, versionKey, watchexecVersion)
	}
	return nil
}
