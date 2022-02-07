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
	"sort"

	"github.com/Masterminds/semver"
)

// ResolveVersion finds the largest version in a list of semantic versions that satisifies the
// provided constraint. If no version in the list statisfies the constraint it returns an error.
func ResolveVersion(constraint string, versions []string) (string, error) {
	if constraint == "" {
		// use the latest version if no constraint was provided
		constraint = "*"
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", err
	}

	semvers := make([]*semver.Version, len(versions))
	for i, version := range versions {
		s, err := semver.NewVersion(version)
		if err != nil {
			return "", err
		}
		semvers[i] = s
	}

	// Sort in descending order so that the first version in the list to satisify a constraint will be
	// the highest possible version.
	sort.Sort(sort.Reverse(semver.Collection(semvers)))
	for _, s := range semvers {
		if c.Check(s) {
			return s.String(), nil
		}
	}

	return "", fmt.Errorf("failed to resolve version matching: %v", c)
}

// IsExactSemver returns true if a given string is valid semantic version.
func IsExactSemver(constraint string) bool {
	_, err := semver.NewVersion(constraint)
	return err == nil
}
