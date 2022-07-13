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

package nodejs

import (
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/Masterminds/semver"
)

const (
	// PackageLock is the name of the npm lock file.
	PackageLock = "package-lock.json"
	// NPMShrinkwrap is the name of the npm shrinkwrap file.
	NPMShrinkwrap = "npm-shrinkwrap.json"
)

var (
	// minPruneVersion is the first npm version that supports the prune command.
	minPruneVersion = semver.MustParse("5.7.0")
	// minNpmCIVersion is the first npm version that suports the ci command.
	minNpmCIVersion = semver.MustParse("6.14.0")
)

// RequestedNPMVersion returns any customer provided NPM version constraint configured in the
// "engines" section of the package.json file in the given application dir.
func RequestedNPMVersion(dir string) (string, error) {
	pjs, err := ReadPackageJSONIfExists(dir)
	if err != nil || pjs == nil || pjs.Engines.NPM == "" {
		return "", err
	}
	version, err := resolvePackageVersion("npm", pjs.Engines.NPM)
	if err != nil {
		gcp.InternalErrorf("fetching npm metadata: %v", err)
	}
	return version, nil
}

// EnsureLockfile returns the name of the lockfile, generating a package-lock.json if necessary.
func EnsureLockfile(ctx *gcp.Context) (string, error) {
	npmShrinkwrapExists, err := ctx.FileExists(NPMShrinkwrap)
	if err != nil {
		return "", err
	}
	// npm prefers npm-shrinkwrap.json, see https://docs.npmjs.com/cli/shrinkwrap.
	if npmShrinkwrapExists {
		return NPMShrinkwrap, nil
	}
	pkgLockExists, err := ctx.FileExists(PackageLock)
	if err != nil {
		return "", err
	}
	if !pkgLockExists {
		ctx.Logf("Generating %s.", PackageLock)
		ctx.Warnf("*** Improve build performance by generating and committing %s.", PackageLock)
		if _, err := ctx.Exec([]string{"npm", "install", "--package-lock-only", "--quiet"}, gcp.WithUserAttribution); err != nil {
			return "", err
		}
	}
	return PackageLock, nil
}

// NPMInstallCommand returns the correct install command based on the version of Node.js. By default
// we prefer "npm ci" because it handles transitive dependencies determinstically. See the NPM docs:
// https://docs.npmjs.com/cli/v6/commands/npm-ci
func NPMInstallCommand(ctx *gcp.Context) (string, error) {
	npmVer, err := npmVersion(ctx)
	if err != nil {
		return "", err
	}
	version, err := semver.NewVersion(npmVer)
	if err != nil {
		return "", gcp.InternalErrorf("parsing npm version: %v", err)
	}
	// HACK: For backwards compatibility with old versions of npm always use `npm install`.
	if version.LessThan(minNpmCIVersion) {
		return "install", nil
	}
	return "ci", nil
}

// npmVersion returns the version of NPM installed in the system.
var npmVersion = func(ctx *gcp.Context) (string, error) {
	result, err := ctx.Exec([]string{"npm", "--version"})
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}

// SupportsNPMPrune returns true if the version of npm installed in the system supports the prune
// command.
func SupportsNPMPrune(ctx *gcp.Context) (bool, error) {
	npmVer, err := npmVersion(ctx)
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(npmVer)
	if err != nil {
		return false, gcp.InternalErrorf("parsing npm version: %v", err)
	}
	return !version.LessThan(minPruneVersion), nil
}
