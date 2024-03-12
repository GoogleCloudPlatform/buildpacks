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

// Package nodejs contains Node.js buildpack library code.
package nodejs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v2"
)

const (
	// EnvNodeEnv is the name of the NODE_ENV environment variable.
	EnvNodeEnv = "NODE_ENV"
	// EnvDevelopment represents a NODE_ENV development value.
	EnvDevelopment = "development"
	// EnvProduction represents a NODE_ENV production value.
	EnvProduction = "production"
	// EnvNodeVersion can be used to specify the version of Node.js is used for an app.
	EnvNodeVersion = "GOOGLE_NODEJS_VERSION"

	nodeVersionKey    = "node_version"
	dependencyHashKey = "dependency_hash"
)

// semVer11 is the smallest possible semantic version with major version 11.
var semVer11 = semver.MustParse("11.0.0")

var (
	cachedPackageJSONs = map[string]*PackageJSON{}
)
var possibleLockfileFilenames = []string{"pnpm-lock.yaml", "yarn.lock", "npm-shrinkwrap.json", "package-lock.json"}

type packageEnginesJSON struct {
	Node string `json:"node"`
	NPM  string `json:"npm"`
	Yarn string `json:"yarn"`
	PNPM string `json:"pnpm"`
}

const (
	// ScriptBuild is the name of npm build scripts.
	ScriptBuild = "build"
	// ScriptGCPBuild is the name of "gcp-build" scripts.
	ScriptGCPBuild = "gcp-build"
)

// PackageJSON represents the contents of a package.json file.
type PackageJSON struct {
	Main            string             `json:"main"`
	Type            string             `json:"type"`
	Version         string             `json:"version"`
	Engines         packageEnginesJSON `json:"engines"`
	Scripts         map[string]string  `json:"scripts"`
	Dependencies    map[string]string  `json:"dependencies"`
	DevDependencies map[string]string  `json:"devDependencies"`
}

// NpmLockfile represents the contents of a lock file generated with npm.
type NpmLockfile struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
}

// PnpmLockfile represents the contents of a lock file generated with pnpm.
type PnpmLockfile struct {
	Dependencies map[string]struct {
		Version string `yaml:"version"`
	} `yaml:"dependencies"`
}

// ReadPackageJSONIfExists returns deserialized package.json from the given dir. If the provided dir
// does not contain a package.json file it returns nil. Empty dir string uses the current working
// directory.
func ReadPackageJSONIfExists(dir string) (*PackageJSON, error) {
	f := filepath.Join(dir, "package.json")
	rawpjs, err := ioutil.ReadFile(f)
	if os.IsNotExist(err) {
		// Return an empty struct if the file doesn't exist (null object pattern).
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading package.json: %v", err)
	}

	var pjs PackageJSON
	if err := json.Unmarshal(rawpjs, &pjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling package.json: %v", err)
	}
	return &pjs, nil
}

// HasGCPBuild returns true if the given package.json file includes a "gcp-build" script.
func HasGCPBuild(p *PackageJSON) bool {
	return HasScript(p, ScriptGCPBuild)
}

// HasScript returns true if the given package.json file defines a script with the given name.
func HasScript(p *PackageJSON, name string) bool {
	if p == nil {
		return false
	}
	_, ok := p.Scripts[name]
	return ok
}

// HasDevDependencies returns true if the given directory contains a package.json file that lists
// more one or more devDependencies.
func HasDevDependencies(p *PackageJSON) bool {
	return p != nil && len(p.DevDependencies) > 0
}

// RequestedNodejsVersion returns any customer provided Node.js version constraint by inspecting the
// environment and the package.json.
func RequestedNodejsVersion(ctx *gcp.Context, pjs *PackageJSON) (string, error) {
	if version := os.Getenv(EnvNodeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", EnvNodeVersion, version)
		return version, nil
	}
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	if pjs == nil {
		return "", nil
	}
	return pjs.Engines.Node, nil
}

// nodeVersion returns the installed version of Node.js.
// It can be overridden for testing.
var nodeVersion = func(ctx *gcp.Context) (string, error) {
	result, err := ctx.Exec([]string{"node", "-v"})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// isPreNode11 returns true if the installed version of Node.js is
// v10.x.x or older.
func isPreNode11(ctx *gcp.Context) (bool, error) {
	nodeVer, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(nodeVer)
	if err != nil {
		return false, gcp.InternalErrorf("failed to detect valid Node.js version %s: %v", version, err)
	}
	return version.LessThan(semVer11), nil
}

// NodeEnv returns the value of NODE_ENV or `production`.
func NodeEnv() string {
	nodeEnv := os.Getenv(EnvNodeEnv)
	if nodeEnv == "" {
		nodeEnv = EnvProduction
	}
	return nodeEnv
}

// CheckOrClearCache checks whether cached dependencies exist and match. If they do not match, the
// layer is cleared and the layer metadata is updated with the new cache key.
func CheckOrClearCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentNodeVersion, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
	opts = append(opts, cache.WithStrings(currentNodeVersion))
	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey, opts...)
	if err != nil {
		return false, err
	}

	if cached {
		return true, nil
	}

	if err := ctx.ClearLayer(l); err != nil {
		return false, fmt.Errorf("clearing layer: %v", err)
	}

	// Update the layer metadata.
	cache.Add(ctx, l, dependencyHashKey, hash)
	ctx.SetMetadata(l, nodeVersionKey, currentNodeVersion)

	return false, nil
}

// SkipSyntaxCheck returns true if we should skip checking the user's function file for syntax errors
// if it is impacted by https://github.com/GoogleCloudPlatform/functions-framework-nodejs/issues/407.
func SkipSyntaxCheck(ctx *gcp.Context, file string, pjs *PackageJSON) (bool, error) {
	nodeVer, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(nodeVer)
	if err != nil {
		return false, gcp.InternalErrorf("failed to detect valid Node.js version %s: %v", version, err)
	}
	if version.Major() != 16 {
		return false, nil
	}
	if strings.HasSuffix(file, ".mjs") {
		return true, nil
	}
	return (pjs != nil && pjs.Type == "module"), nil
}

// IsNodeJS8Runtime returns true when the GOOGLE_RUNTIME is nodejs8. This will be
// true when using GCF or GAE with nodejs8. This function is useful for some
// legacy behavior in GCF.
func IsNodeJS8Runtime() bool {
	return os.Getenv(env.Runtime) == "nodejs8"
}

func versionFromPnpmLock(rawPackageLock []byte, pkg string) (string, error) {
	var lockfile PnpmLockfile
	if err := yaml.Unmarshal(rawPackageLock, &lockfile); err != nil {
		return "", gcp.InternalErrorf("parsing pnpm lock file: %w", err)
	}
	return strings.Split(lockfile.Dependencies[pkg].Version, "(")[0], nil
}

func versionFromYarnLock(rawPackageLock []byte, pjs *PackageJSON, pkg string) (string, error) {
	// yarn requires custom parsing since it has a custom format
	// this logic works for both yarn classic and berry
	for _, dependency := range strings.Split(string(rawPackageLock[:]), "\n\n") {
		if strings.Contains(dependency, pkg+"@") && strings.Contains(dependency, pjs.Dependencies[pkg]) {
			for _, line := range strings.Split(dependency, "\n") {
				if strings.Contains(line, "version") {
					return strings.Trim(strings.Fields(line)[1], `"`), nil
				}
			}
		}
	}
	return "", gcp.InternalErrorf("parsing yarn file")
}

func versionFromNpmLock(rawPackageLock []byte, pkg string) (string, error) {
	var lockfile NpmLockfile
	if err := json.Unmarshal(rawPackageLock, &lockfile); err != nil {
		return "", gcp.InternalErrorf("parsing lock file: %w", err)
	}
	return lockfile.Packages["node_modules/"+pkg].Version, nil
}

// Version tries to get the concrete package version used based on lock file,
// returns error if no lock file is found or is misshapen
func Version(ctx *gcp.Context, pjs *PackageJSON, pkg string) (string, error) {
	for _, filename := range possibleLockfileFilenames {
		filePath := filepath.Join(ctx.ApplicationRoot(), filename)
		rawPackageLock, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		switch filename {
		case "pnpm-lock.yaml":
			return versionFromPnpmLock(rawPackageLock, pkg)
		case "yarn.lock":
			return versionFromYarnLock(rawPackageLock, pjs, pkg)
		case "npm-shrinkwrap.json", "package-lock.json":
			return versionFromNpmLock(rawPackageLock, pkg)
		}
	}

	return "", gcp.UserErrorf("No lock file found, please run npm install to generate one")
}
