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
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// Match against ruby string example: ruby 2.6.7p450
var rubyVersionRe = regexp.MustCompile(`^\s*ruby\s+([^p^\s]+)(p\d+)?\s*$`)

const defaultVersion = "3.0.0"

// DetectVersion detects ruby version from the environment, Gemfile.lock, gems.locked, or falls
// back to a default version.
func DetectVersion(ctx *gcp.Context) (string, error) {
	versionFromEnv := os.Getenv(env.RuntimeVersion)
	// The two lock files have the same format for Ruby version
	lockFiles := []string{"Gemfile.lock", "gems.locked"}

	for _, lockFileName := range lockFiles {

		path := filepath.Join(ctx.ApplicationRoot(), lockFileName)
		pathExists, err := ctx.FileExists(path)
		if err != nil {
			return "", err
		}
		if pathExists {

			file, err := os.Open(path)
			if err != nil {
				return "", err
			}

			defer file.Close()
			lockedVersion, err := lockFileVersion(lockFileName, file)

			if err != nil {
				return "", err
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
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, versionFromEnv)
		return versionFromEnv, nil
	}

	return defaultVersion, nil
}

// lockFileVersion extacts the version number from Gemfile.lock or gems.locked, returns an error in
// case the version string is malformed.
func lockFileVersion(fileName string, r io.Reader) (string, error) {
	const token = "RUBY VERSION"

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		if scanner.Text() == token {
			// Read the next line once the token is found
			if !scanner.Scan() {
				break
			}

			version := scanner.Text()

			matches := rubyVersionRe.FindStringSubmatch(version)
			if len(matches) > 1 {
				return matches[1], nil
			}

			return "", gcp.UserErrorf("Invalid ruby version in %s: %q", fileName, version)
		}
	}

	return "", nil
}
