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
)

const (
	// PackageLock is the name of the npm lock file.
	PackageLock = "package-lock.json"
	// NPMShrinkwrap is the name of the npm shrinkwrap file.
	NPMShrinkwrap = "npm-shrinkwrap.json"
)

// EnsureLockfile returns the name of the lockfile, generating a package-lock.json if necessary.
func EnsureLockfile(ctx *gcp.Context) string {
	// npm prefers npm-shrinkwrap.json, see https://docs.npmjs.com/cli/shrinkwrap.
	if ctx.FileExists(NPMShrinkwrap) {
		return NPMShrinkwrap
	}
	if !ctx.FileExists(PackageLock) {
		ctx.Logf("Generating %s.", PackageLock)
		ctx.Warnf("*** Improve build performance by generating and committing %s.", PackageLock)
		ctx.Exec([]string{"npm", "install", "--package-lock-only", "--quiet"}, gcp.WithUserAttribution)
	}
	return PackageLock
}

// NPMInstallCommand returns the correct install commmand based on the version of Node.js.
func NPMInstallCommand(ctx *gcp.Context) string {
	// HACK: For backwards compatibility on App Engine Node.js 10, always use `npm install`.
	if strings.HasPrefix(strings.TrimSpace(NodeVersion(ctx)), "v10.") {
		return "install"
	}
	return "ci"
}
