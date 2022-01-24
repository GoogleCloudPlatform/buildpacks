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
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}

	// The two lock files have the same format for Ruby version
	lockFiles := []string{"Gemfile.lock", "gems.locked"}

	for _, lockFileName := range lockFiles {

		path := filepath.Join(ctx.ApplicationRoot(), lockFileName)
		if ctx.FileExists(path) {

			file, err := os.Open(path)
			if err != nil {
				return "", err
			}

			defer file.Close()
			return lockFileVersion(lockFileName, file)
		}
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

	return defaultVersion, nil
}
