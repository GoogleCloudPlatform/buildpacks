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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v2"
)

var (
	yarnURL    = "https://yarnpkg.com/downloads/%[1]s/yarn-v%[1]s.tar.gz"
	yarn2URL   = "https://repo.yarnpkg.com/%s/packages/yarnpkg-cli/bin/yarn.js"
	version2   = semver.MustParse("2.0.0")
	versionKey = "version"
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

// UseFrozenLockfile returns an true if the environment supporte Yarn's --frozen-lockfile flag. This
// is a hack to maintain backwards compatibility on App Engine Node.js 10 and older.
func UseFrozenLockfile(ctx *gcp.Context) (bool, error) {
	oldNode, err := isPreNode11(ctx)
	return !oldNode, err
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

// HasYarnWorkspacePlugin returns true if this project has Yarn2's workspaces plugin installed.
func HasYarnWorkspacePlugin(ctx *gcp.Context) (bool, error) {
	res, err := ctx.Exec([]string{"yarn", "plugin", "runtime"})
	if err != nil {
		return false, err
	}
	return strings.Contains(res.Stdout, "plugin-workspace-tools"), nil
}

// detectYarnVersion determines the version of Yarn that should be installed in a Node.js project
// by examining the "engines.yarn" and "packageManager" constraints specified in package.json and comparing it against all
// published versions in the NPM registry, if both exist "engines.yarn" will take precedence.
// If the package.json does not include "engines.yarn" or "packageManager" it
// returns the latest stable version available.
// TODO(b/338411091) create a shared packagejson util library and refactor out a generic detect
// package manager version function.
func detectYarnVersion(pjs *PackageJSON) (string, error) {
	if pjs == nil || (pjs.Engines.Yarn == "" && pjs.PackageManager == "") {
		version, err := latestPackageVersion("yarn")
		if err != nil {
			return "", gcp.InternalErrorf("fetching available Yarn versions: %w", err)
		}
		return version, nil
	}
	var requestedVersion string
	if pjs.Engines.Yarn != "" {
		requestedVersion = pjs.Engines.Yarn
	} else {
		packageManagerName, packageManagerVersion, err := parsePackageManager(pjs.PackageManager)
		if err != nil {
			return "", err
		}
		if packageManagerName != "yarn" {
			return "", gcp.UserErrorf("yarn was detected but %s is set in the packageManager package.json field.", packageManagerName)
		}
		requestedVersion = packageManagerVersion
	}
	version, err := resolvePackageVersion("yarn", requestedVersion)
	if err != nil {
		return "", gcp.UserErrorf("finding Yarn version that matched %q: %w", requestedVersion, err)
	}
	return version, nil
}

// InstallYarnLayer installs Yarn in the given layer if it is not already cached.
func InstallYarnLayer(ctx *gcp.Context, yarnLayer *libcnb.Layer, pjs *PackageJSON) error {
	layerName := yarnLayer.Name
	version, err := detectYarnVersion(pjs)
	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(yarnLayer, versionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("Yarn cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(yarnLayer); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install yarn in layer.
		ctx.Logf("Installing Yarn v%s", version)
		if err := InstallYarn(ctx, yarnLayer.Path, version); err != nil {
			return err
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(yarnLayer, versionKey, version)
	// We need to update the path here to ensure the version we just installed take precendence over
	// anything pre-installed in the base image.
	if err := ctx.Setenv("PATH", filepath.Join(yarnLayer.Path, "bin")+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}

// InstallYarn downloads a given version of Yarn into the provided directory.
func InstallYarn(ctx *gcp.Context, dir, version string) error {
	v, err := semver.NewVersion(version)
	if err != nil {
		gcp.UserErrorf("parsing yarn version %q: %v", version, err)
	}
	if v.LessThan(version2) {
		archiveURL := fmt.Sprintf(yarnURL, version)
		stripComponents := 1
		return fetch.Tarball(archiveURL, dir, stripComponents)
	}

	yarnPath := filepath.Join(dir, "bin", "yarn")
	if err = os.MkdirAll(filepath.Dir(yarnPath), 0755); err != nil {
		return gcp.InternalErrorf("creating directory %q: %v", filepath.Dir(yarnPath), err)
	}
	out, err := os.OpenFile(yarnPath, os.O_CREATE|os.O_RDWR, os.FileMode(0777))
	if err != nil {
		return gcp.InternalErrorf("creating file %q: %v", yarnPath, err)
	}
	defer out.Close()
	binURL := fmt.Sprintf(yarn2URL, version)
	if err = fetch.GetURL(binURL, out); err != nil {
		return err
	}
	return nil
}
