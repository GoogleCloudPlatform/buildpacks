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

// Implements python/pip buildpack.
// The pip buildpack installs dependencies using pip.
package main

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName = "pip"
	cacheName = "pipcache"
)

// metadata represents metadata stored for a dependencies layer.
type metadata struct {
	PythonVersion   string `toml:"python_version"`
	DependencyHash  string `toml:"dependency_hash"`
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("requirements.txt") {
		ctx.OptOut("requirements.txt not found")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName)
	cl := ctx.Layer(cacheName)

	cached, meta, err := python.CheckCache(ctx, l, cache.WithFiles("requirements.txt"))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(layerName)
		return nil
	}
	ctx.CacheMiss(layerName)

	// Install modules in requirements.txt.
	ctx.Logf("Running pip install.")
	ctx.Exec([]string{"python3", "-m", "pip", "install", "--upgrade", "-r", "requirements.txt", "-t", l.Root}, gcp.WithEnv("PIP_CACHE_DIR="+cl.Root))

	ctx.PrependPathSharedEnv(l, "PYTHONPATH", l.Root)

	// Check for broken dependencies.
	ctx.Logf("Checking for incompatible dependencies.")
	checkDeps := ctx.Exec([]string{"python3", "-m", "pip", "check"}, gcp.WithEnv("PYTHONPATH="+l.Root+":"+os.Getenv("PYTHONPATH")))
	if checkDeps.ExitCode != 0 {
		return fmt.Errorf("incompatible dependencies installed: %q", checkDeps.Stdout)
	}

	ctx.WriteMetadata(l, &meta, layers.Build, layers.Cache, layers.Launch)
	ctx.WriteMetadata(cl, nil, layers.Cache)
	return nil
}
