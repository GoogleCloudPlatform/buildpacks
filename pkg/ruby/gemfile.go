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

package ruby

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/Masterminds/semver"
)

// Match against ruby string example: ruby 2.6.7p450
var rubyVersionRe = regexp.MustCompile(`^\s*ruby\s+([^p^\s]+)(p\d+)?\s*$`)

// ParseRubyVersion extracts the version number from Gemfile.lock or gems.locked, returns an error in
// case the version string is malformed.
func ParseRubyVersion(path string) (string, error) {
	version, err := readLineAfter(path, "RUBY VERSION")
	if err != nil {
		return "", err
	}
	if version == "" {
		return "", nil
	}

	matches := rubyVersionRe.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", gcp.UserErrorf("parsing ruby version %q", version)
}

// ParseBundlerVersion extacts the version of bundler from Gemfile.lock or gems.locked,
// returns an error in case the version string is malformed.
func ParseBundlerVersion(path string) (string, error) {
	version, err := readLineAfter(path, "BUNDLED WITH")
	if err != nil {
		return "", err
	}
	if version == "" {
		return "", nil
	}

	semver, err := semver.NewVersion(strings.TrimSpace(version))
	if err != nil {
		return "", gcp.UserErrorf("parsing bundler version %q: %v", version, err)
	}

	return fmt.Sprintf("%d.%d.%d", semver.Major(), semver.Minor(), semver.Patch()), nil
}

func readLineAfter(path string, token string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == token {
			// Read the next line once the token is found
			if !scanner.Scan() {
				break
			}

			return scanner.Text(), nil
		}
	}

	return "", nil
}
