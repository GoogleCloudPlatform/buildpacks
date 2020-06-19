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
	"github.com/buildpack/libbuildpack/layers"
)

const (
	dateFormat = time.RFC3339Nano
	// expirationTime is an arbitrary amount of time of 1 day to refresh the cache layer.
	expirationTime = time.Duration(time.Hour * 24)
)

// Metadata represents metadata stored for a dependencies layer.
type Metadata struct {
	PythonVersion   string `toml:"python_version"`
	DependencyHash  string `toml:"dependency_hash"`
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

// Version returns the installed version of Python.
func Version(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"python3", "--version"})
	return strings.TrimSpace(result.Stderr)
}

// CheckCache checks whether cached dependencies exist and match.
func CheckCache(ctx *gcp.Context, l *layers.Layer, opts ...cache.Option) (bool, *Metadata, error) {
	currentPythonVersion := Version(ctx)
	opts = append(opts, cache.WithStrings(currentPythonVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, nil, fmt.Errorf("computing dependency hash: %v", err)
	}

	var meta Metadata
	ctx.ReadMetadata(l, &meta)

	expired := checkCacheExpiration(ctx, &meta)

	// Perform install, skipping if the dependency hash matches existing metadata.
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", meta.DependencyHash)
	if currentDependencyHash == meta.DependencyHash && !expired {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, &meta, nil
	}

	if meta.DependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}

	ctx.ClearLayer(l)

	ctx.Logf("Installing application dependencies.")
	// Update the layer metadata.
	meta.DependencyHash = currentDependencyHash
	meta.PythonVersion = currentPythonVersion
	meta.ExpiryTimestamp = time.Now().Add(expirationTime).Format(dateFormat)

	return false, &meta, nil
}

// checkCacheExpiration returns true when the cache is past expiration.
func checkCacheExpiration(ctx *gcp.Context, meta *Metadata) bool {
	t := time.Now()
	if meta.ExpiryTimestamp != "" {
		var err error
		t, err = time.Parse(dateFormat, meta.ExpiryTimestamp)
		if err != nil {
			ctx.Debugf("Could not parse expiration date %q, assuming now: %v", meta.ExpiryTimestamp, err)
		}
	}
	if t.After(time.Now()) {
		return false
	}
	return true
}
