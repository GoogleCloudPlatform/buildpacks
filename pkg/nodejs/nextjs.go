// Copyright 2023 Google LLC
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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v2"
)

var (
	// nextJsVersionKey is the metadata key used to store the nextjs build adaptor version in the nextjs layer.
	nextJsVersionKey = "version"
)
var possibleLockfileFilenames = []string{"pnpm-lock.yaml", "yarn.lock", "npm-shrinkwrap.json", "package-lock.json"}

// NpmLockfile represents the contents of a lock file generated with npm
type NpmLockfile struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
}

// PnpmLockfile represents the contents of a lock file generated with pnpm
type PnpmLockfile struct {
	Dependencies map[string]struct {
		Version string `yaml:"version"`
	} `yaml:"dependencies"`
}

// InstallNextJsBuildAdaptor installs the nextjs build adaptor in the given layer if it is not already cached.
func InstallNextJsBuildAdaptor(ctx *gcp.Context, njsl *libcnb.Layer, njsVersion string) error {
	layerName := njsl.Name
	version, err := detectNextjsAdaptorVersion(njsVersion)

	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(njsl, nextJsVersionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("nextjs adaptor cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(njsl); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install nextjs adaptor in layer.
		ctx.Logf("Installing nextjs adaptor %s", version)
		if err := downloadNextJsAdaptor(ctx, njsl.Path, version); err != nil {
			return gcp.InternalErrorf("downloading nextjs adapter: %w", err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(njsl, nextJsVersionKey, version)
	return nil
}

// detectNextjsAdaptorVersion determines the version of Nextjs that is needed by a nextjs project
func detectNextjsAdaptorVersion(njsVersion string) (string, error) {
	version, err := semver.StrictNewVersion(njsVersion)
	if err != nil {
		return "", gcp.InternalErrorf("parsing nextjs version: %w", err)
	}
	// match major + minor versions with the Nextjs version
	adapterVersion := strconv.FormatUint(version.Major(), 10) + "." + strconv.FormatUint(version.Minor(), 10)
	return adapterVersion, nil
}

// downloadNextJsAdaptor downloads the Nextjs build adaptor into the provided directory.
func downloadNextJsAdaptor(ctx *gcp.Context, dirPath string, version string) error {
	if _, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-nextjs@" + version}); err != nil {
		ctx.Logf("Failed to install nextjs adaptor version: %s. Falling back to latest", version)
		if _, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-nextjs@latest"}); err != nil {
			return gcp.InternalErrorf("installing nextjs adaptor: %w", err)
		}
	}
	return nil
}

// OverrideNextjsBuildScript overrides the build script to be the Nextjs build script
func OverrideNextjsBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-nextjs-build", njsl.Path))
}

// Version tries to get the concrete nextjs version used based on lock file, returns error if no lock file is found or is mishappen
func Version(ctx *gcp.Context, pjs *PackageJSON) (string, error) {
	for _, filename := range possibleLockfileFilenames {
		filePath := filepath.Join(ctx.ApplicationRoot(), filename)
		rawPackageLock, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		if filename == "pnpm-lock.yaml" {
			var lockfile PnpmLockfile
			if err := yaml.Unmarshal(rawPackageLock, &lockfile); err != nil {
				return "", gcp.InternalErrorf("parsing pnpm lock file: %w", err)
			}
			return strings.Split(lockfile.Dependencies["next"].Version, "(")[0], nil
		}

		if filename == "yarn.lock" {
			// yarn requires custom parsing since it has a custom format
			// this logic works for both yarn classic and berry
			for _, dependency := range strings.Split(string(rawPackageLock[:]), "\n\n") {
				if strings.Contains(dependency, "next@") && strings.Contains(dependency, pjs.Dependencies["next"]) {
					for _, line := range strings.Split(dependency, "\n") {
						if strings.Contains(line, "version") {
							return strings.Trim(strings.Fields(line)[1], `"`), nil
						}
					}
				}
			}
			return "", gcp.InternalErrorf("parsing yarn file")
		}

		if filename == "npm-shrinkwrap.json" || filename == "package-lock.json" {
			var lockfile NpmLockfile
			if err := json.Unmarshal(rawPackageLock, &lockfile); err != nil {
				return "", gcp.InternalErrorf("parsing lock file: %w", err)
			}
			return lockfile.Packages["node_modules/next"].Version, nil
		}
	}

	return "", gcp.UserErrorf("No lock file found, please run npm install to generate one")
}
