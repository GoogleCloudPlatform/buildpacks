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
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
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
	if result := runtime.CheckOverride(ctx, "python"); result != nil {
		return result, nil
	}

	if !ctx.HasAtLeastOne("*.py") {
		return gcp.OptOut("no .py files found"), nil
	}
	return gcp.OptIn("found .py files"), nil
}

func legacyInstallPython(ctx *gcp.Context, layer *libcnb.Layer) (bool, error) {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return false, fmt.Errorf("determining runtime version: %w", err)
	}
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     pythonLayer,
		Metadata: map[string]interface{}{"version": version},
		Launch:   true,
		Build:    true,
	})

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(layer, versionKey)
	if version == metaVersion {
		ctx.CacheHit(pythonLayer)
		return true, nil
	}
	ctx.CacheMiss(pythonLayer)
	ctx.ClearLayer(layer)

	archiveURL := fmt.Sprintf(pythonURL, version)
	code, err := ctx.HTTPStatus(archiveURL)
	if err != nil {
		return false, err
	}
	if code != http.StatusOK {
		return false, gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	ctx.Logf("Installing Python v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s", archiveURL, layer.Path)
	ctx.Exec([]string{"bash", "-c", command})

	ctx.SetMetadata(layer, versionKey, version)
	return false, nil
}

func buildFn(ctx *gcp.Context) error {
	layer := ctx.Layer(pythonLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	var isCached bool
	var err error
	// Use GOOGLE_PYTHON_VERSION to enable installing from the experimental tarball hosting service.
	if version := os.Getenv(versionEnv); version != "" {
		isCached, err = runtime.InstallTarballIfNotCached(ctx, runtime.Python, version, layer)
	} else {
		isCached, err = legacyInstallPython(ctx, layer)
	}

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
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, v)
		return v, nil
	}
	if ctx.FileExists(versionFile) {
		raw := ctx.ReadFile(versionFile)
		v := strings.TrimSpace(string(raw))
		if v != "" {
			ctx.Logf("Using runtime version from %s: %s", versionFile, v)
			return v, nil
		}
		return "", gcp.UserErrorf("%s exists but does not specify a version", versionFile)
	}
	// Intentionally no user-attributed becase the URL is provided by Google.
	v := ctx.Exec([]string{"curl", "--fail", "--show-error", "--silent", "--location", versionURL}).Stdout
	ctx.Logf("Using latest runtime version: %s", v)
	return v, nil
}
