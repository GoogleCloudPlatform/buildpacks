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
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	pythonLayer = "python"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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
	sysconfig, _ := ctx.Exec([]string{"/layers/google.python.runtime/python/bin/python3", "-m", "sysconfig"}, gcp.WithUserAttribution)
	leftIdentifier := "exec_prefix = \""
	rightIdentifier := "\""
	start := strings.Index(sysconfig.Stdout, leftIdentifier)
	end := strings.Index(sysconfig.Stdout[start:], rightIdentifier)
	execPrefix := sysconfig.Stdout[start+len(leftIdentifier) : start+end+len(leftIdentifier)]
	result, _ := ctx.Exec([]string{
		"grep",
		"-rlI",
		execPrefix,
		layer.Path,
	}, gcp.WithUserAttribution)
	paths := strings.Split(result.Stdout, "\n")
	for _, path := range paths {
		ctx.Exec([]string{
			"sed",
			"-i",
			"s|" + execPrefix + "|" + layer.Path + "|g",
			path,
		}, gcp.WithUserAttribution)
	}

	// Force stdout/stderr streams to be unbuffered so that log messages appear immediately in the logs.
	layer.LaunchEnvironment.Default("PYTHONUNBUFFERED", "TRUE")

	ctx.Logf("Upgrading pip to the latest version and installing build tools")
	path := filepath.Join(layer.Path, "bin/python3")
	if _, err := ctx.Exec([]string{path, "-m", "pip", "install", "--upgrade", "pip", "setuptools==v64.0.0", "wheel"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	return nil
}
