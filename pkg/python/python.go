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

// Package python contains Python buildpack library code.
package python

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	dateFormat = time.RFC3339Nano
	// expirationTime is an arbitrary amount of time of 1 day to refresh the cache layer.
	expirationTime = time.Duration(time.Hour * 24)

	pythonVersionKey   = "python_version"
	dependencyHashKey  = "dependency_hash"
	expiryTimestampKey = "expiry_timestamp"

	cacheName = "pipcache"
)

// Version returns the installed version of Python.
func Version(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"python3", "--version"})
	return strings.TrimSpace(result.Stdout)
}

// InstallRequirements installs dependencies from the given requirements file.
// The function creates a layer for pip cache and returns a path to the site-packages
// directory that is added to PYTHONPATH by lifecycle for subsequent buildpacks and att
// launch time.
func InstallRequirements(ctx *gcp.Context, l *libcnb.Layer, req string) (string, error) {
	l.Cache = true // The layer always needs to be cache=true for the logic below to work.
	cl := ctx.Layer(cacheName, gcp.CacheLayer)

	// Use the `site` package to get version-specific path to site-packages.
	result := ctx.Exec([]string{"python3", "-m", "site", "--user-site"}, gcp.WithEnv("PYTHONUSERBASE="+l.Path))
	path := strings.TrimSpace(result.Stdout)
	l.SharedEnvironment.PrependPath("PYTHONPATH", path)

	// Check if we can use the cached-layer as is without reinstalling dependencies.
	cached, err := checkCache(ctx, l, cache.WithFiles(req))
	if err != nil {
		return "", fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(l.Name)
		return path, nil
	}
	ctx.CacheMiss(l.Name)

	// pip install --target has several subtle issues:
	// We cannot use --upgrade: https://github.com/pypa/pip/issues/8799.
	// We also cannot _not_ use --upgrade, see the requirements_bin_conflict acceptance test.
	//
	// Instead, we use Python per-user site-packages (https://www.python.org/dev/peps/pep-0370/)
	// to generate the version-specific path where package are installed combined with --prefix.
	//
	// For backwards compatibility with base image updates, we cannot use --user because the
	// base image activates a virtual environment which is not compatible with the --user flag.

	ctx.Exec([]string{
		"python3", "-m", "pip", "install",
		"--requirement", req,
		"--upgrade",
		"--upgrade-strategy", "only-if-needed",
		"--no-warn-script-location", // bin is added at run time by lifecycle.
		"--ignore-installed",        // Some dependencies may be in the build image but not run image.
		"--prefix", l.Path,
	},
		gcp.WithEnv("PIP_CACHE_DIR="+cl.Path),
		gcp.WithUserAttribution)

	return path, nil
}

// checkCache checks whether cached dependencies exist, match, and have not expired.
func checkCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentPythonVersion := Version(ctx)
	opts = append(opts, cache.WithStrings(currentPythonVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, fmt.Errorf("computing dependency hash: %v", err)
	}

	metaDependencyHash := ctx.GetMetadata(l, dependencyHashKey)
	// Check cache expiration to pick up new versions of dependencies that are not pinned.
	expired := cacheExpired(ctx, l)

	// Perform install, skipping if the dependency hash matches existing metadata.
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", metaDependencyHash)
	if currentDependencyHash == metaDependencyHash && !expired {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}

	ctx.ClearLayer(l)

	ctx.Logf("Installing application dependencies.")
	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, currentDependencyHash)
	ctx.SetMetadata(l, pythonVersionKey, currentPythonVersion)
	ctx.SetMetadata(l, expiryTimestampKey, time.Now().Add(expirationTime).Format(dateFormat))

	return false, nil
}

// cacheExpired returns true when the cache is past expiration.
func cacheExpired(ctx *gcp.Context, l *libcnb.Layer) bool {
	t := time.Now()
	expiry := ctx.GetMetadata(l, expiryTimestampKey)
	if expiry != "" {
		var err error
		t, err = time.Parse(dateFormat, expiry)
		if err != nil {
			ctx.Debugf("Could not parse expiration date %q, assuming now: %v", expiry, err)
		}
	}
	return !t.After(time.Now())
}
