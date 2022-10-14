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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const defaultVersion = "3.0.0"

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
			return lockedVersion, err
		}
	}

	if versionFromEnv != "" {
		ctx.Logf(
			"Using runtime version from environment variable %s: %s", env.RuntimeVersion, versionFromEnv)
		return versionFromEnv, nil
	}

	return defaultVersion, nil
}

// IsRuby25 returns true if the build environment has Ruby 2.5.x installed.
func IsRuby25(ctx *gcp.Context) bool {
	return strings.HasPrefix(os.Getenv(env.RuntimeVersion), "2.5")
}
