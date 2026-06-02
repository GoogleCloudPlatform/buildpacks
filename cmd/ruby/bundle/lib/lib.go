// Copyright 2025 Google LLC
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
package lib

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName         = "gems"
	dependencyHashKey = "dependency_hash"
	rubyVersionKey    = "ruby_version"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
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

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
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

	localGemsDir := filepath.Join(".bundle", "gems")
	localBinDir := filepath.Join(".bundle", "bin")

	// 1. LOCK PHASE (with Maker override)
	if cap := ctx.Capability(ruby.BundleLockerCapability); cap != nil {
		l, ok := cap.(ruby.BundleLocker)
		if !ok {
			return gcp.InternalErrorf("capability %q must implement BundleLocker", ruby.BundleLockerCapability)
		}
		if err := l.Lock(ctx); err != nil {
			return err
		}
	} else {
		// Default Lockfile preparation
		if err := ruby.PrepareLockfile(ctx, localGemsDir, "development test", []string{"x86_64-linux", "ruby"}); err != nil {
			return err
		}
		// Buildpack clears the config after locking to avoid cache hash pollution
		if err := ctx.RemoveAll(".bundle"); err != nil {
			return err
		}
	}

	// 2. MAKER SHORTCUT EXIT
	// If in Maker mode, run the custom local installation and exit immediately, bypassing layers/caching.
	if cap := ctx.Capability(ruby.BundleInstallerCapability); cap != nil {
		i, ok := cap.(ruby.BundleInstaller)
		if !ok {
			return gcp.InternalErrorf("capability %q must implement BundleInstaller", ruby.BundleInstallerCapability)
		}
		return i.Install(ctx)
	}

	// 3. STANDARD BUILDPACK FLOW (Layers & Caching)
	deps, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	bundleOutput := filepath.Join(deps.Path, ".bundle")

	cached, err := checkCache(ctx, deps, cache.WithFiles(lockFile))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}

	if cached {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)

		// Install the bundle locally into .bundle/gems
		bCfg := ruby.BundleConfig{
			Deployment: true,
			Frozen:     true,
		}
		env := []string{"NOKOGIRI_USE_SYSTEM_LIBRARIES=1", "MALLOC_ARENA_MAX=2", "LANG=C.utf8"}
		if err := ruby.InstallAndSymlink(ctx, localGemsDir, localBinDir, "development test", bCfg, env); err != nil {
			return err
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
