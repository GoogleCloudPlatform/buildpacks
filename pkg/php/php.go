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

// Package php contains PHP buildpack library code.
package php

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	// composerJSON is the name of the Composer package descriptor file.
	composerJSON = "composer.json"
	// composerLock is the name of the Composer lock file.
	composerLock = "composer.lock"
	// Vendor is the name of the Composer vendor directory.
	Vendor = "vendor"
)

type composerScriptsJSON struct {
	GCPBuild string `json:"gcp-build"`
}

// ComposerJSON represents the contents of a composer.json file.
type ComposerJSON struct {
	Require map[string]string   `json:"require"`
	Scripts composerScriptsJSON `json:"scripts"`
}

// Metadata represents metadata stored for a dependencies layer.
type Metadata struct {
	PHPVersion     string `toml:"php_version"`
	DependencyHash string `toml:"dependency_hash"`
}

// ReadComposerJSON returns the deserialized composer.json from the given dir. Empty dir uses the current working directory.
func ReadComposerJSON(dir string) (*ComposerJSON, error) {
	f := filepath.Join(dir, composerJSON)
	rawcjs, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %v", composerJSON, err)
	}

	var cjs ComposerJSON
	if err := json.Unmarshal(rawcjs, &cjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling %s: %v", composerJSON, err)
	}
	return &cjs, nil
}

// version returns the installed version of PHP.
func version(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"php", "-r", "echo PHP_VERSION;"})
	return result.Stdout
}

// checkCache checks whether cached dependencies exist and match.
func checkCache(ctx *gcp.Context, l *layers.Layer, opts ...cache.Option) (bool, *Metadata, error) {
	currentPHPVersion := version(ctx)
	opts = append(opts, cache.WithStrings(currentPHPVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, nil, fmt.Errorf("computing dependency hash: %v", err)
	}

	var meta Metadata
	ctx.ReadMetadata(l, &meta)

	// Perform install, skipping if the dependency hash matches existing metadata.
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", meta.DependencyHash)
	if currentDependencyHash == meta.DependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, &meta, nil
	}

	if meta.DependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	ctx.Logf("Installing application dependencies.")
	// Update the layer metadata.
	meta.DependencyHash = currentDependencyHash
	meta.PHPVersion = currentPHPVersion

	return false, &meta, nil
}

// composerInstall runs `composer install` with the given flags.
func composerInstall(ctx *gcp.Context, flags []string) {
	cmd := append([]string{"composer", "install"}, flags...)
	ctx.ExecUser(cmd)
}

// ComposerInstall runs `composer install`, using the cache iff a lock file is present.
// It creates a layer, so it returns the layer so that the caller may further modify it
// if they desire.
func ComposerInstall(ctx *gcp.Context, cacheTag string) (*layers.Layer, error) {
	// We don't install dev dependencies (i.e. we pass --no-dev to composer) because doing so has caused
	// problems for customers in the past. For more information see these links:
	//   https://github.com/GoogleCloudPlatform/php-docs-samples/issues/736
	//   https://github.com/GoogleCloudPlatform/runtimes-common/pull/763
	//   https://github.com/GoogleCloudPlatform/runtimes-common/commit/6c4970f609d80f9436ac58ae272cfcc6bcd57143
	flags := []string{"--no-dev", "--no-progress", "--no-suggest", "--no-interaction"}

	ctx.RemoveAll(Vendor)
	l := ctx.Layer("composer")
	layerVendor := filepath.Join(l.Root, Vendor)

	// If there's no composer.lock then don't attempt to cache. We'd have to cache using composer.json,
	// which could result in outdated dependencies if the version constraints in composer.json resolve
	// to newer versions in the future.
	if !ctx.FileExists(composerLock) {
		ctx.Logf("*** Improve build performance by generating and committing %s.", composerLock)
		composerInstall(ctx, flags)
		return l, nil
	}

	cached, meta, err := checkCache(ctx, l, cache.WithStrings(composerLock))
	if err != nil {
		return l, fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)

		// PHP expects the vendor/ directory to be in the application directory.
		ctx.Exec([]string{"cp", "--archive", layerVendor, Vendor})
	} else {
		ctx.CacheMiss(cacheTag)
		// Clear layer so we don't end up with outdated dependencies (e.g. something was removed from composer.json).
		ctx.ClearLayer(l)
		composerInstall(ctx, flags)

		// Ensure vendor exists even if no dependencies were installed.
		ctx.MkdirAll(Vendor, 0755)
		ctx.Exec([]string{"cp", "--archive", Vendor, layerVendor})
	}

	ctx.WriteMetadata(l, &meta, layers.Cache)
	return l, nil
}

// ComposerRequire runs `composer require` with the given packages. It expects packages to
// be specified as `composer require` would expect them on the command line, for example
// "myorg/mypackage:^0.7". It does no caching.
func ComposerRequire(ctx *gcp.Context, packages []string) {
	cmd := append([]string{"composer", "require", "--no-progress", "--no-suggest", "--no-interaction"}, packages...)
	ctx.ExecUser(cmd)
}
