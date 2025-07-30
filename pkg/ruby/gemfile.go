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

// AddBundledGemsIfNecessary checks the ruby version and adds bundled gems to the Gemfile if they are not present.
func AddBundledGemsIfNecessary(ctx *gcp.Context, rubyVersion, gemfilePath string) error {
	v, err := semver.NewVersion(rubyVersion)
	if err != nil {
		return fmt.Errorf("parsing ruby version %q: %w", rubyVersion, err)
	}
	// This logic should only apply to Ruby 3.4 and greater.
	ruby34, _ := semver.NewVersion("3.4.0")
	if v.LessThan(ruby34) {
		return nil
	}

	bundledGems := []string{
		"rexml", "rss", "webrick", "net-ftp", "net-imap", "net-pop", "net-smtp", "matrix", "prime", "open-uri",
	}

	content, err := ctx.ReadFile(gemfilePath)
	if err != nil {
		return fmt.Errorf("reading Gemfile: %w", err)
	}

	var gemsToAdd []string
	for _, gem := range bundledGems {
		// A simple check to see if the gem is already in the Gemfile.
		// This is not perfect, as it could match comments or parts of other gem names,
		// but it's a good enough heuristic for now.
		if !strings.Contains(string(content), fmt.Sprintf("gem '%s'", gem)) &&
			!strings.Contains(string(content), fmt.Sprintf("gem \"%s\"", gem)) {
			gemsToAdd = append(gemsToAdd, gem)
		}
	}

	if len(gemsToAdd) > 0 {
		ctx.Logf("Adding bundled gems to Gemfile: %s", strings.Join(gemsToAdd, ", "))
		f, err := os.OpenFile(gemfilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening Gemfile for appending: %w", err)
		}
		defer f.Close()

		for _, gem := range gemsToAdd {
			if _, err := f.WriteString(fmt.Sprintf("\ngem '%s'", gem)); err != nil {
				return fmt.Errorf("appending gem %q to Gemfile: %w", gem, err)
			}
		}
	}

	return nil
}
