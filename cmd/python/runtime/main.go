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

// Implements python/runtime buildpack.
// The runtime buildpack installs the Python runtime.
package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/flex"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	pythonLayer = "python"
)

var execPrefixRegex = regexp.MustCompile(`exec_prefix\s*=\s*"([^"]+)`)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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

func buildFn(ctx *gcp.Context) error {
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
	// replace python sysconfig variable prefix from "/opt/python" to "/layers/google.python.runtime/python/" which is the layer.Path
	// python is installed in /layers/google.python.runtime/python/ for unified builder,
	// while the python downloaded from debs is installed in "/opt/python".
	sysconfig, _ := ctx.Exec([]string{filepath.Join(layer.Path, "bin/python3"), "-m", "sysconfig"})
	execPrefix, err := parseExecPrefix(sysconfig.Stdout)
	if err != nil {
		return err
	}
	result, _ := ctx.Exec([]string{
		"grep",
		"-rlI",
		execPrefix,
		layer.Path,
	})
	paths := strings.Split(result.Stdout, "\n")
	for _, path := range paths {
		ctx.Exec([]string{
			"sed",
			"-i",
			"s|" + execPrefix + "|" + layer.Path + "|g",
			path,
		})
	}
	// Set the PYTHONHOME for flex apps because of uwsgi
	if env.IsFlex() {
		layer.LaunchEnvironment.Default("PYTHONHOME", layer.Path)
	}
	// Force stdout/stderr streams to be unbuffered so that log messages appear immediately in the logs.
	layer.LaunchEnvironment.Default("PYTHONUNBUFFERED", "TRUE")
	return nil
}

func parseExecPrefix(sysconfig string) (string, error) {
	match := execPrefixRegex.FindStringSubmatch(sysconfig)
	if len(match) < 2 {
		return "", fmt.Errorf("determining Python exec prefix: %v", match)
	}
	return match[1], nil
}
