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

// Package version provides utility methods for working with semantic versions.
package version

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/Masterminds/semver"
)

var skipKeywords = []string{"deprecated", "public-image", "latest"}

type resolveParams struct {
	noSanitize bool
}

// ResolveVersionOption configures ResolveVersion.
type ResolveVersionOption func(o *resolveParams)

// WithoutSanitization indicates the return value should not have any prefix trimmed or 0s appended.
var WithoutSanitization = func(o *resolveParams) {
	o.noSanitize = true
}

// ResolveVersion finds the largest version in a list of semantic versions that satisfies the
// provided constraint. If no version in the list satisfies the constraint it returns an error.
func ResolveVersion(constraint string, versions []string, opts ...ResolveVersionOption) (string, error) {
	params := resolveParams{}
	for _, o := range opts {
		o(&params)
	}
	if constraint == "" {
		// use the latest version if no constraint was provided
		constraint = "*"
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", err
	}

	semvers := []*semver.Version(nil)
	for _, version := range versions {
		if shouldSkipVersion(version, skipKeywords) {
			continue
		}
		s, err := semver.NewVersion(version)
		if err != nil {
			return "", buildererror.Errorf(buildererror.StatusInternal, "failed to parse version %q: %v", version, err)
		}
		semvers = append(semvers, s)
	}

	// Sort in descending order so that the first version in the list to satisfy a constraint will be
	// the highest possible version.
	sortVersionsDesc(semvers)
	for _, s := range semvers {
		if c.Check(s) {
			if params.noSanitize {
				return s.Original(), nil
			}
			return s.String(), nil
		}
	}

	return "", fmt.Errorf("failed to resolve version matching: %s against %v", c.String(), semvers)
}

func sortVersionsDesc(versions []*semver.Version) {
	slices.SortStableFunc(versions, func(a, b *semver.Version) int {
		if a.Equal(b) {
			return strings.Compare(b.Metadata(), a.Metadata())
		}
		return b.Compare(a)
	})
}

func shouldSkipVersion(version string, keywords []string) bool {
	if IsReleaseCandidate(version) {
		return true
	}
	for _, keyword := range keywords {
		if strings.HasPrefix(strings.ToLower(version), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// IsExactSemver returns true if a given string is valid semantic version.
func IsExactSemver(constraint string) bool {
	if strings.Count(constraint, ".") != 2 {
		// The constraint must include the major, minor, and patch segments to be exact. By default,
		// semver.NewVersion will set these to zero so we must validate this separately.
		return false
	}
	_, err := semver.NewVersion(constraint)
	return err == nil
}

// IsReleaseCandidate returns true if given string is a RC candidate version.
// When launching a new runtime, we need to test with RC candidate which will eventually be replaced
// by a stable candidate. Till then, we will support these unstable releases in the QA for testing.
// Example python rc version - 3.12.0rc1
// Example dotnet rc version - 8.0.100-rc.1
// Example php rc version - 8.3.0RC4
// Example ruby rc version - 3.2.0-rc1
func IsReleaseCandidate(constraint string) bool {
	m := regexp.MustCompile(`(\d+)\.(\d+)\.(.*)(rc|RC)(.*)`)
	return m.MatchString(constraint)
}
