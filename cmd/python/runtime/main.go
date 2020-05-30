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
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	pythonLayer = "python"
	pythonURL   = "https://storage.googleapis.com/gcp-buildpacks/python/python-%s.tar.gz"
	// TODO(b/148375706): Add mapping for stable/beta versions.
	versionURL  = "https://storage.googleapis.com/gcp-buildpacks/python/latest.version"
	versionFile = ".python-version"
)

// metadata represents metadata stored for a runtime layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	runtime.CheckOverride(ctx, "python")

	if !ctx.HasAtLeastOne("*.py") {
		ctx.OptOut("No *.py files found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta metadata
	l := ctx.Layer(pythonLayer)
	ctx.ReadMetadata(l, &meta)
	if version == meta.Version {
		ctx.CacheHit(pythonLayer)
		return nil
	}
	ctx.CacheMiss(pythonLayer)
	ctx.ClearLayer(l)

	archiveURL := fmt.Sprintf(pythonURL, version)
	if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
	}

	ctx.Logf("Installing Python v%s", version)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s", archiveURL, l.Root)
	ctx.Exec([]string{"bash", "-c", command})

	ctx.Logf("Upgrading pip to the latest version and installing build tools")
	path := filepath.Join(l.Root, "bin/python3")
	ctx.Exec([]string{path, "-m", "pip", "install", "--upgrade", "pip", "setuptools", "wheel"})

	meta.Version = version
	ctx.WriteMetadata(l, meta, layers.Build, layers.Cache, layers.Launch)

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    pythonLayer,
		Version: version,
	})
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
	v := ctx.Exec([]string{"curl", "--silent", versionURL}).Stdout
	ctx.Logf("Using latest runtime version: %s", v)
	return v, nil
}
