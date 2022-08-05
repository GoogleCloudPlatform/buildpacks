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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	pythonLayer = "python"
	pythonURL   = "https://storage.googleapis.com/gcp-buildpacks/python/python-%s.tar.gz"
	// TODO(b/148375706): Add mapping for stable/beta versions.
	versionURL  = "https://storage.googleapis.com/gcp-buildpacks/python/latest.version"
	versionFile = ".python-version"
	versionKey  = "version"
	versionEnv  = "GOOGLE_PYTHON_VERSION"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("python"); result != nil {
		return result, nil
	}

	atLeastOne, err := ctx.HasAtLeastOne("*.py")
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
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", pythonLayer, err)
	}
	ver, err := runtimeVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	if _, err := runtime.InstallTarballIfNotCached(ctx, runtime.Python, ver, layer); err != nil {
		return err
	}
	// Force stdout/stderr streams to be unbuffered so that log messages appear immediately in the logs.
	layer.LaunchEnvironment.Default("PYTHONUNBUFFERED", "TRUE")

	ctx.Logf("Upgrading pip to the latest version and installing build tools")
	path := filepath.Join(layer.Path, "bin/python3")
	if _, err := ctx.Exec([]string{path, "-m", "pip", "install", "--upgrade", "pip", "setuptools", "wheel"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	return nil
}

func runtimeVersion(ctx *gcp.Context) (string, error) {
	v1, err := versionFromEnv(ctx)
	if err != nil {
		return "", err
	}
	v2, err := versionFromFile(ctx)
	if err != nil {
		return "", err
	}
	if v1 != "" && v2 != "" && v1 != v2 {
		return "", gcp.UserErrorf("python version %s from %s file and %s from environment variable are inconsistent, pick one of them or set them to the same value",
			v1, versionFile, v2)
	}
	if v1 != "" {
		return v1, nil
	}
	if v2 != "" {
		return v2, nil
	}

	// This will use the highest listed at https://dl.google.com/runtimes/python/version.json.
	ctx.Logf("Python version not specified, using the latest available version.")
	return "*", nil
}

func versionFromEnv(ctx *gcp.Context) (string, error) {
	v1 := os.Getenv(versionEnv)
	v2 := os.Getenv(env.RuntimeVersion)

	if v1 != "" && v2 != "" && v1 != v2 {
		return "", gcp.UserErrorf("%s=%s and %s=%s are inconsistent, pick one of them or set them to the same value",
			versionEnv, v1, env.RuntimeVersion, v2)
	}

	if v1 != "" {
		ctx.Logf("Using Python version from %s: %s", versionEnv, v1)
		return v1, nil
	}
	if v2 != "" {
		ctx.Logf("Using Python version from %s: %s", env.RuntimeVersion, v2)
		return v2, nil
	}
	return "", nil
}

func versionFromFile(ctx *gcp.Context) (string, error) {
	versionFileExists, err := ctx.FileExists(versionFile)
	if err != nil {
		return "", err
	}
	if versionFileExists {
		raw, err := ctx.ReadFile(versionFile)
		if err != nil {
			return "", err
		}
		v := strings.TrimSpace(string(raw))
		if v != "" {
			ctx.Logf("Using Python version from %s: %s", versionFile, v)
			return v, nil
		}
		return "", gcp.UserErrorf("%s exists but does not specify a version", versionFile)
	}
	return "", nil
}
