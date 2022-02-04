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
	"io/ioutil"
	"path/filepath"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"gopkg.in/yaml.v2"
)

const (
	// YarnLock is the name of the yarn lock file.
	YarnLock = "yarn.lock"
)

type yarn2Lock struct {
	Metadata struct {
		Version string `yaml:"version"`
	} `yaml:"__metadata"`
}

// YarnInstallCmd returns an appropriate command for installing Yarn dependencies for a given environment.
func YarnInstallCmd(ctx *gcp.Context, yarn2, pnpMode bool) ([]string, error) {
	if yarn2 {
		cmd := []string{"yarn", "install", "--immutable"}
		if pnpMode {
			// In Plug'n'Play mode all dependencies must be included in the Yarn cache. The --immutable-cache
			// option will abort the install with an error if anything is missing or out of date.
			cmd = append(cmd, "--immutable-cache")
		}
		return cmd, nil
	}

	// Setting --production=false causes the devDependencies to be installed regardless of the
	// NODE_ENV value. The allows the customer's lifecycle hooks to access to them. We purge the
	// devDependencies from the final app.
	cmd := []string{"yarn", "install", "--non-interactive", "--prefer-offline", "--production=false"}

	// HACK: For backwards compatibility on App Engine Node.js 10 and older, skip using `--frozen-lockfile`.
	if isOldNode, err := isPreNode11(ctx); err != nil {
		return nil, err
	} else if !isOldNode {
		cmd = append(cmd, "--frozen-lockfile")
	}
	return cmd, nil
}

// IsYarn2 detects whether the given lockfile was generated with Yarn 2.
func IsYarn2(rootDir string) (bool, error) {
	data, err := ioutil.ReadFile(filepath.Join(rootDir, YarnLock))
	if err != nil {
		return false, gcp.InternalErrorf("reading yarn.lock: %v", err)
	}

	var manifest yarn2Lock

	if err := yaml.Unmarshal(data, &manifest); err != nil {
		// In Yarn1, yarn.lock was not necessarily valid YAML.
		return false, nil
	}
	// After Yarn2, yarn.lock files contain a __metadata.version field.
	return manifest.Metadata.Version != "", nil
}

// IsYarnPNP returns true if the project is using Yarn2's Plug'n'Play feature.
func IsYarnPNP(ctx *gcp.Context) bool {
	return ctx.FileExists(ctx.ApplicationRoot(), ".yarn", "cache")
}

// HasYarnWorkspacePlugin returns true if this project has Yarn2's workspaces plugin installed.
func HasYarnWorkspacePlugin(ctx *gcp.Context) bool {
	res := ctx.Exec([]string{"yarn", "plugin", "runtime"})
	return strings.Contains(res.Stdout, "plugin-workspace-tools")
}

// DetectYarnVersion determines the version of Yarn that should be installed in a Node.js project
// by examining the "engines.yarn" constraint specified in package.json and comparing it against all
// published versions in the NPM registry. If the package.json does not include "engines.yarn" it
// returns the latest stable version available.
func DetectYarnVersion(applicationRoot string) (string, error) {
	requested, err := requestedYarnVersion(applicationRoot)
	if err != nil {
		return "", err
	}
	if requested == "" {
		version, err := latestPackageVersion("yarn")
		if err != nil {
			return "", gcp.InternalErrorf("fetching available Yarn versions: %w", err)
		}
		return version, nil
	}

	version, err := resolvePackageVersion("yarn", requested)
	if err != nil {
		return "", gcp.UserErrorf("finding Yarn version that matched %q: %w", requested, err)
	}
	return version, nil
}

// requestedYarnVersion returns the Yarn version specified in the "engines.yarn" section of the
// project's package.json.
func requestedYarnVersion(applicationRoot string) (string, error) {
	pjs, err := ReadPackageJSON(applicationRoot)
	if err != nil {
		return "", err
	}
	return pjs.Engines.Yarn, nil
}
