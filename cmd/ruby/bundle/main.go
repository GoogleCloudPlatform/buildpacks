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

// Implements ruby/bundle buildpack.
// The bundle buildpack installs dependencies using bundle.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	layerName         = "gems"
	dependencyHashKey = "dependency_hash"
	rubyVersionKey    = "ruby_version"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.FileExists("Gemfile") {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	if ctx.FileExists("gems.rb") {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	return gcp.OptOut("no Gemfile or gems.rb found"), nil
}

func buildFn(ctx *gcp.Context) error {
	var lockFile string
	hasGemfile := ctx.FileExists("Gemfile")
	hasGemsRB := ctx.FileExists("gems.rb")
	if hasGemfile {
		if hasGemsRB {
			ctx.Warnf("Gemfile and gems.rb both exist. Using Gemfile.")
		}
		if !ctx.FileExists("Gemfile.lock") {
			return gcp.Errorf(gcp.StatusFailedPrecondition, "Could not find Gemfile.lock file in your app. Please make sure your bundle is up to date before deploying.")
		}
		lockFile = "Gemfile.lock"
	} else if hasGemsRB {
		if !ctx.FileExists("gems.locked") {
			return gcp.Errorf(gcp.StatusFailedPrecondition, "Could not find gems.locked file in your app. Please make sure your bundle is up to date before deploying.")
		}
		lockFile = "gems.locked"
	}

	// Remove any user-provided local bundle config and cache that can interfere with the build process.
	ctx.RemoveAll(".bundle")

	// Ensure the GCP runtime platform is present in the lockfile. This is needed for Bundler >= 2.2, in case the user's lockfile is specific to a different platform.
	ctx.Exec([]string{"bundle", "lock", "--add-platform", "x86_64-linux"}, gcp.WithUserAttribution)
	ctx.Exec([]string{"bundle", "lock", "--add-platform", "ruby"}, gcp.WithUserAttribution)

	deps := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)

	// This layer directory contains the files installed by bundler into the application .bundle directory
	bundleOutput := filepath.Join(deps.Path, ".bundle")

	cached, err := checkCache(ctx, deps, cache.WithFiles(lockFile))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)

		localGemsDir := filepath.Join(".bundle", "gems")
		localBinDir := filepath.Join(".bundle", "bin")

		// Install the bundle locally into .bundle/gems
		ctx.Exec([]string{"bundle", "config", "--local", "deployment", "true"}, gcp.WithUserAttribution)
		ctx.Exec([]string{"bundle", "config", "--local", "frozen", "true"}, gcp.WithUserAttribution)
		ctx.Exec([]string{"bundle", "config", "--local", "without", "development test"}, gcp.WithUserAttribution)
		ctx.Exec([]string{"bundle", "config", "--local", "path", localGemsDir}, gcp.WithUserAttribution)
		ctx.Exec([]string{"bundle", "install"}, gcp.WithUserAttribution)

		// Find any gem-installed binary directory and symlink as a static path
		foundBinDirs := ctx.Glob(".bundle/gems/ruby/*/bin")
		if len(foundBinDirs) > 1 {
			return fmt.Errorf("unexpected multiple gem bin dirs: %v", foundBinDirs)
		} else if len(foundBinDirs) == 1 {
			ctx.Symlink(filepath.Join(ctx.ApplicationRoot(), foundBinDirs[0]), localBinDir)
		}

		// Move the built .bundle directory into the layer
		ctx.RemoveAll(bundleOutput)
		ctx.Exec([]string{"mv", ".bundle", bundleOutput}, gcp.WithUserTimingAttribution)
	}

	// Always link local .bundle directory to the actual installation stored in the layer.
	ctx.Symlink(bundleOutput, ".bundle")

	return nil
}

// checkCache checks whether cached dependencies exist and match.
func checkCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentRubyVersion := ctx.Exec([]string{"ruby", "-v"}).Stdout
	opts = append(opts, cache.WithStrings(currentRubyVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, fmt.Errorf("computing dependency hash: %v", err)
	}

	// Perform install, skipping if the dependency hash matches existing metadata.
	metaDependencyHash := ctx.GetMetadata(l, dependencyHashKey)
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", metaDependencyHash)
	if currentDependencyHash == metaDependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	ctx.Logf("Installing application dependencies.")

	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, currentDependencyHash)
	ctx.SetMetadata(l, rubyVersionKey, currentRubyVersion)

	return false, nil
}
