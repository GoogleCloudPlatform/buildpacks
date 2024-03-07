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
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/Masterminds/semver"
)

const (
	// PackageLock is the name of the npm lock file.
	PackageLock = "package-lock.json"
	// NPMShrinkwrap is the name of the npm shrinkwrap file.
	NPMShrinkwrap = "npm-shrinkwrap.json"
	// GoogleNodeRunScriptsEnv is the env var that can be used to configure a list of package.json
	// scripts that should be run during the build process.
	GoogleNodeRunScriptsEnv = "GOOGLE_NODE_RUN_SCRIPTS"
	// nodejsNPMBuildEnv is an env var that enables running `npm run build` by default.
	nodejsNPMBuildEnv = "GOOGLE_EXPERIMENTAL_NODEJS_NPM_BUILD_ENABLED"
	// VendorNpmDeps for vendoring npm dependencies
	VendorNpmDeps = "GOOGLE_VENDOR_NPM_DEPENDENCIES"
	// AppHostingBuildEnv is the env var that contains the build script to run for nextjs apps
	AppHostingBuildEnv = "APPHOSTING_BUILD"
)

var (
	// minPruneVersion is the first npm version that supports the prune command.
	minPruneVersion = semver.MustParse("5.7.0")
	// minNpmCIVersion is the first npm version that suports the ci command.
	minNpmCIVersion = semver.MustParse("6.14.0")
)

// RequestedNPMVersion returns any customer provided NPM version constraint configured in the
// "engines" section of the package.json file in the given application dir.
func RequestedNPMVersion(pjs *PackageJSON) (string, error) {
	if pjs == nil || pjs.Engines.NPM == "" {
		return "", nil
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
	// b/236758688: For backwards compatibility on GAE & GCF Node.js 10 and older, always use `npm install`.
	if env.IsGAE() || env.IsGCF() {
		isOldNode, err := isPreNode11(ctx)
		if err != nil {
			return "", err
		}
		if isOldNode {
			return "install", nil
		}
	}
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

// DetermineBuildCommands returns a list of "npm run" commands to be executed during the build
// and a bool representing whether this is a "custom build" (user-specified build scripts)
// or a system build step (default build behavior).
//
// Users can specify npm scripts to run in three ways, with the following order of precedence:
// 1. APPHOSTING_BUILD env var
// 2. GOOGLE_NODE_RUN_SCRIPTS env var
// 3. "gcp-build" script in package.json
// 4. "build" script in package.json
func DetermineBuildCommands(pjs *PackageJSON, pkgTool string) (cmds []string, isCustomBuild bool) {
	appHostingBuildScript, appHostingBuildScriptPresent := os.LookupEnv(AppHostingBuildEnv)
	if appHostingBuildScriptPresent {
		return []string{appHostingBuildScript}, true
	}

	envScript, envScriptPresent := os.LookupEnv(GoogleNodeRunScriptsEnv)
	if envScriptPresent {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmGoogleNodeRunScriptsUsageCounterID).Increment(1)
		// Setting `GOOGLE_NODE_RUN_SCRIPTS=` preserves legacy behavior where "npm run build" was NOT
		// run, even though "build" was provided.
		if strings.TrimSpace(envScript) == "" {
			return []string{}, true
		}

		scripts := strings.Split(envScript, ",")
		for _, s := range scripts {
			cmds = append(cmds, runCommand(pkgTool, s))
		}

		return cmds, true
	}

	if HasGCPBuild(pjs) {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmGcpBuildUsageCounterID).Increment(1)
		if gcpBuild := pjs.Scripts[ScriptGCPBuild]; strings.TrimSpace(gcpBuild) == "" {
			return []string{}, true
		}
		return []string{runCommand(pkgTool, "gcp-build")}, true
	}

	if HasScript(pjs, ScriptBuild) {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.NpmBuildUsageCounterID).Increment(1)
		if build := pjs.Scripts[ScriptBuild]; strings.TrimSpace(build) == "" {
			return []string{}, false
		}
		return []string{runCommand(pkgTool, "build")}, false
	}

	return []string{}, false
}

// IsUsingVendoredDependencies returns true if the builder should be using the vendored dependencies.
func IsUsingVendoredDependencies() bool {
	val, _ := env.IsPresentAndTrue(VendorNpmDeps)
	return val
}

func runCommand(pkgTool, command string) string {
	return fmt.Sprintf("%s run %s", pkgTool, strings.TrimSpace(command))
}

// DefaultStartCommand returns the default command that should be used to configure the application
// web process if the user has not explicitly configured one. The algorithm follows the conventions
// of Nodejs package.json files: https://docs.npmjs.com/cli/v10/configuring-npm/package-json#main
// 1. if script.start is specified return `npm run start`
// 2. if the project contains server.js `npm run start`
// 3. if main is specified `node ${pjs.main}`
// 4. otherwise `node index.js“
func DefaultStartCommand(ctx *gcp.Context, pjs *PackageJSON) ([]string, error) {
	if pjs == nil {
		return []string{"node", "index.js"}, nil
	}
	if angularStart := ExtractAngularStartCommand(pjs); angularStart != "" {
		return strings.Fields(angularStart), nil
	}
	if _, ok := pjs.Scripts["start"]; ok {
		return []string{"npm", "run", "start"}, nil
	}
	if nuxt, err := NuxtStartCommand(ctx); err != nil || nuxt != nil {
		return nuxt, err
	}
	exists, err := ctx.FileExists(ctx.ApplicationRoot(), "server.js")
	if err != nil {
		return nil, err
	}
	if exists {
		return []string{"npm", "run", "start"}, nil
	}
	if pjs.Main != "" {
		return []string{"node", pjs.Main}, nil
	}
	return []string{"node", "index.js"}, nil
}
