// Copyright 2022 Google LLC
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

// Package ruby contains Ruby buildpack library code.
package ruby

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/Masterminds/semver"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const defaultVersion = "3.3.*"

// RubyVersionKey is the environment variable name used to store the Ruby version installed.
const RubyVersionKey = "build_ruby_version"

// DetectVersion detects ruby version from the environment, Gemfile.lock, gems.locked, or falls
// back to a default version.
func DetectVersion(ctx *gcp.Context) (string, error) {
	versionFromEnv := os.Getenv(env.RuntimeVersion)
	// The two lock files have the same format for Ruby version
	lockFiles := []string{"Gemfile.lock", "gems.locked"}

	// If environment is GAE or GCF, skip lock file validation.
	// App Engine specific validation is done in a different buildpack.
	if env.IsGAE() || env.IsGCF() {
		if versionFromEnv != "" {
			ctx.Logf(
				"Using runtime version from environment variable %s: %s", env.RuntimeVersion, versionFromEnv)
			return versionFromEnv, nil
		}
	}

	versionFromRubyVersion, err := getVersionFromRubyVersion(ctx)
	if err != nil {
		return "", err
	}
	if versionFromEnv != "" && versionFromRubyVersion != "" && versionFromRubyVersion != versionFromEnv {
		return "", gcp.UserErrorf(
			"There is a conflict between Ruby versions specified in .ruby-version file and the %s environment variable. "+
				"Please resolve the conflict by choosing only one way to specify the ruby version.",
			env.RuntimeVersion)
	}

	for _, lockFileName := range lockFiles {

		path := filepath.Join(ctx.ApplicationRoot(), lockFileName)
		pathExists, err := ctx.FileExists(path)
		if err != nil {
			return "", err
		}
		if pathExists {
			lockedVersion, err := ParseRubyVersion(path)

			if err != nil {
				return "", gcp.UserErrorf("Error %q in: %s", err, lockFileName)
			}

			// Lockfile doesn't contain a ruby version, so we can move on
			if lockedVersion == "" {
				break
			}

			// Bundler doesn't allow us to override a version of ruby if it's locked in the lock file
			// The env will still be useful if a project doesn't lock ruby version or doesn't use bundler
			if versionFromEnv != "" && lockedVersion != versionFromEnv {
				return "", gcp.UserErrorf(
					"Ruby version %q in %s can't be overriden to %q using %s environment variable",
					lockedVersion, lockFileName, versionFromEnv, env.RuntimeVersion)
			}
			if versionFromRubyVersion != "" && lockedVersion != versionFromRubyVersion {
				return "", gcp.UserErrorf(
					"There is a conflict between the Ruby version %q in %s and %q in .ruby-version file."+
						"Please resolve the conflict by choosing only one way to specify the ruby version.",
					lockedVersion, lockFileName, versionFromRubyVersion)
			}
			return lockedVersion, err
		}
	}

	if versionFromEnv != "" {
		ctx.Logf(
			"Using runtime version from environment variable %s: %s", env.RuntimeVersion, versionFromEnv)
		return versionFromEnv, nil
	}
	if versionFromRubyVersion != "" {
		ctx.Logf(
			"Using runtime version from .ruby-version file: %s", versionFromRubyVersion)
		return versionFromRubyVersion, nil
	}

	return defaultVersion, nil
}

// IsRuby25 returns true if the build environment has Ruby 2.5.x installed.
func IsRuby25(ctx *gcp.Context) bool {
	return strings.HasPrefix(os.Getenv(RubyVersionKey), "2.5")
}

// SupportsBundler1 returns true if the installed Ruby version is compatible with Bundler 1.
// Bundler 1 breaks with Ruby 3.2. This functions returns true for all versions older than 3.2.
func SupportsBundler1(ctx *gcp.Context) (bool, error) {
	rubyVersion, err := semver.NewVersion(os.Getenv(RubyVersionKey))
	if err != nil {
		return false, err
	}
	ruby32Version, _ := semver.NewVersion("3.2.0")
	return rubyVersion.LessThan(ruby32Version), nil
}

// NeedsRailsAssetPrecompile detects if asset precompilation is required in a Ruby on Rails app.
func NeedsRailsAssetPrecompile(ctx *gcp.Context) (bool, error) {
	isRailsApp, err := ctx.FileExists("bin", "rails")
	if err != nil {
		return false, fmt.Errorf("finding bin/rails: %w", err)
	}
	if !isRailsApp {
		return false, nil
	}

	assetsExists, err := ctx.FileExists("app", "assets")
	if err != nil {
		return false, err
	}
	if !assetsExists {
		return false, nil
	}

	manifestExists, err := ctx.FileExists("public", "assets", "manifest.yml")
	if err != nil {
		return false, err
	}
	if manifestExists {
		return false, nil
	}

	matches, err := ctx.Glob("public/assets/manifest-*.json")
	if err != nil {
		return false, fmt.Errorf("finding manifets: %w", err)
	}
	if matches != nil {
		return false, nil
	}

	matches, err = ctx.Glob("public/assets/.sprockets-manifest-*.json")
	if err != nil {
		return false, fmt.Errorf("finding sprockets-manifets: %w", err)
	}
	if matches != nil {
		return false, nil
	}

	return true, nil
}

// Function to get the ruby version from .ruby-version file.
func getVersionFromRubyVersion(ctx *gcp.Context) (string, error) {
	path := filepath.Join(ctx.ApplicationRoot(), ".ruby-version")
	pathExists, err := ctx.FileExists(path)
	if err != nil {
		return "", err
	}
	if pathExists {
		version, err := os.ReadFile(path)
		if err != nil {
			return "", gcp.UserErrorf("Error %q in: %s", err, ".ruby-version")
		}
		return string(version), nil
	}
	return "", nil
}
