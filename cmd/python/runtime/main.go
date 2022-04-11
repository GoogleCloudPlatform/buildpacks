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
	layer, err := ctx.Layer(pythonLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", pythonLayer, err)
	}
	ver, err := runtimeVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	isCached, err := runtime.InstallTarballIfNotCached(ctx, runtime.Python, ver, layer)
	if err != nil {
		return err
	}
	if !isCached {
		// Force stdout/stderr streams to be unbuffered so that log messages appear immediately in the logs.
		layer.LaunchEnvironment.Default("PYTHONUNBUFFERED", "TRUE")

		ctx.Logf("Upgrading pip to the latest version and installing build tools")
		path := filepath.Join(layer.Path, "bin/python3")
		ctx.Exec([]string{path, "-m", "pip", "install", "--upgrade", "pip", "setuptools", "wheel"}, gcp.WithUserAttribution)
	}
	return nil
}

func runtimeVersion(ctx *gcp.Context) (string, error) {
	if v := os.Getenv(versionEnv); v != "" {
		ctx.Logf("Using Python version from %s: %s", versionEnv, v)
		return v, nil
	}
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		ctx.Logf("Using Python version from %s: %s", env.RuntimeVersion, v)
		return v, nil
	}
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
	// This will use the highest listed at https://dl.google.com/runtimes/python/version.json.
	ctx.Logf("Python version not specified, using the test available version.")
	return "*", nil
}
