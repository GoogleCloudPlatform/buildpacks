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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
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
	gemfileExists, err := ctx.FileExists("Gemfile")
	if err != nil {
		return nil, err
	}
	if gemfileExists {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	gemsRbExists, err := ctx.FileExists("gems.rb")
	if err != nil {
		return nil, err
	}
	if gemsRbExists {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	return gcp.OptOut("no Gemfile or gems.rb found"), nil
}

func buildFn(ctx *gcp.Context) error {
	var lockFile string
	hasGemfile, err := ctx.FileExists("Gemfile")
	if err != nil {
		return err
	}
	hasGemsRB, err := ctx.FileExists("gems.rb")
	if err != nil {
		return err
	}
	if hasGemfile {
		if hasGemsRB {
			ctx.Warnf("Gemfile and gems.rb both exist. Using Gemfile.")
		}
		gemfileLockExists, err := ctx.FileExists("Gemfile.lock")
		if err != nil {
			return err
		}
		if !gemfileLockExists {
			return buildererror.Errorf(buildererror.StatusFailedPrecondition, "Could not find Gemfile.lock file in your app. Please make sure your bundle is up to date before deploying.")
		}
		lockFile = "Gemfile.lock"
	} else if hasGemsRB {
		gemsLockedExists, err := ctx.FileExists("gems.locked")
		if err != nil {
			return err
		}
		if !gemsLockedExists {
			return buildererror.Errorf(buildererror.StatusFailedPrecondition, "Could not find gems.locked file in your app. Please make sure your bundle is up to date before deploying.")
		}
		lockFile = "gems.locked"
	}

	// Remove any user-provided local bundle config and cache that can interfere with the build process.
	if err := ctx.RemoveAll(".bundle"); err != nil {
		return err
	}

	deps, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	// This layer directory contains the files installed by bundler into the application .bundle directory
	bundleOutput := filepath.Join(deps.Path, ".bundle")

	cached, err := checkCache(ctx, deps, cache.WithFiles(lockFile))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}

	localGemsDir := filepath.Join(".bundle", "gems")
	localBinDir := filepath.Join(".bundle", "bin")

	// Ensure the GCP runtime platform is present in the lockfile. This is needed for Bundler >= 2.2, in case the user's lockfile is specific to a different platform.
	if _, err := ctx.Exec([]string{"bundle", "config", "--local", "without", "development test"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	if _, err := ctx.Exec([]string{"bundle", "config", "--local", "path", localGemsDir}, gcp.WithUserAttribution); err != nil {
		return err
	}

	// This line will override user provided BUNDLED WITH in the Gemfile.lock
	// It'll use the currently activated bundler version instead
	// This was a change in bundler 2.1+
	// https://github.com/rubygems/rubygems/issues/5683
	if _, err := ctx.Exec([]string{"bundle", "lock", "--add-platform", "x86_64-linux"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	if _, err := ctx.Exec([]string{"bundle", "lock", "--add-platform", "ruby"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	if err := ctx.RemoveAll(".bundle"); err != nil {
		return err
	}

	if cached {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)

		// Install the bundle locally into .bundle/gems
		if _, err := ctx.Exec([]string{"bundle", "config", "--local", "deployment", "true"}, gcp.WithUserAttribution); err != nil {
			return err
		}
		if _, err := ctx.Exec([]string{"bundle", "config", "--local", "frozen", "true"}, gcp.WithUserAttribution); err != nil {
			return err
		}
		if _, err := ctx.Exec([]string{"bundle", "config", "--local", "without", "development test"}, gcp.WithUserAttribution); err != nil {
			return err
		}
		if _, err := ctx.Exec([]string{"bundle", "config", "--local", "path", localGemsDir}, gcp.WithUserAttribution); err != nil {
			return err
		}
		if _, err := ctx.Exec([]string{"bundle", "install"},
			gcp.WithEnv("NOKOGIRI_USE_SYSTEM_LIBRARIES=1", "MALLOC_ARENA_MAX=2", "LANG=C.utf8"), gcp.WithUserAttribution); err != nil {
			return err
		}

		// Find any gem-installed binary directory and symlink as a static path
		foundBinDirs, err := ctx.Glob(".bundle/gems/ruby/*/bin")
		if err != nil {
			return fmt.Errorf("finding bin dirs: %w", err)
		}
		if len(foundBinDirs) > 1 {
			return fmt.Errorf("unexpected multiple gem bin dirs: %v", foundBinDirs)
		} else if len(foundBinDirs) == 1 {
			if err := ctx.Symlink(filepath.Join(ctx.ApplicationRoot(), foundBinDirs[0]), localBinDir); err != nil {
				return err
			}
		}

		// Move the built .bundle directory into the layer
		if err := ctx.RemoveAll(bundleOutput); err != nil {
			return err
		}
		if _, err := ctx.Exec([]string{"mv", ".bundle", bundleOutput}, gcp.WithUserTimingAttribution); err != nil {
			return err
		}
	}

	// Always link local .bundle directory to the actual installation stored in the layer.
	if err := ctx.Symlink(bundleOutput, ".bundle"); err != nil {
		return err
	}

	return nil
}

// checkCache checks whether cached dependencies exist and match.
func checkCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	result, err := ctx.Exec([]string{"ruby", "-v"})
	if err != nil {
		return false, err
	}
	currentRubyVersion := result.Stdout
	opts = append(opts, cache.WithStrings(currentRubyVersion))

	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey, opts...)
	if err != nil {
		return false, err
	}

	if cached {
		return true, nil
	}

	ctx.Logf("Installing application dependencies.")
	cache.Add(ctx, l, dependencyHashKey, hash)
	// Update the layer metadata.
	ctx.SetMetadata(l, rubyVersionKey, currentRubyVersion)

	return false, nil
}
