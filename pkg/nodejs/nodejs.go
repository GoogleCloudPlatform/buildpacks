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
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
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
	// defaultVersionConstraint is used if the project does not provide a Node.js version specifier in
	// their package.json or via an env var. This pins them to the active LTS version, instead of the
	// the latest available version.
	defaultVersionConstraint = "20.*.*"
)

// semVer11 is the smallest possible semantic version with major version 11.
var semVer11 = semver.MustParse("11.0.0")

var (
	cachedPackageJSONs        = map[string]*PackageJSON{}
	possibleLockfileFilenames = []string{"pnpm-lock.yaml", "yarn.lock", "npm-shrinkwrap.json", "package-lock.json"}
	dependencyRegex           = regexp.MustCompile(`\r?\n\r?\n`)
)

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
	// ScriptApphostingBuild is the name of "apphosting-build" scripts.
	ScriptApphostingBuild = "apphosting:build"
)

// PackageJSON represents the contents of a package.json file.
type PackageJSON struct {
	Name            string             `json:"name"`
	Main            string             `json:"main"`
	Type            string             `json:"type"`
	Version         string             `json:"version"`
	Engines         packageEnginesJSON `json:"engines"`
	Scripts         map[string]string  `json:"scripts"`
	Dependencies    map[string]string  `json:"dependencies"`
	DevDependencies map[string]string  `json:"devDependencies"`
	PackageManager  string             `json:"packageManager"`
}

// NpmLockfile represents the contents of a lock file generated with npm.
type NpmLockfile struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
}

// PnpmV6Lockfile represents the contents of a lock file v6 generated with pnpm.
type PnpmV6Lockfile struct {
	Dependencies map[string]struct {
		Version string `yaml:"version"`
	} `yaml:"dependencies"`
	DevDependencies map[string]struct {
		Version string `yaml:"version"`
	} `yaml:"devDependencies"`
}

// PnpmV9Lockfile represents the contents of a lock file v9 generated with pnpm.
type PnpmV9Lockfile struct {
	Importers struct {
		Dot struct {
			Dependencies map[string]struct {
				Version string `yaml:"version"`
			} `yaml:"dependencies"`
			DevDependencies map[string]struct {
				Version string `yaml:"version"`
			} `yaml:"devDependencies"`
		} `yaml:"."`
	} `yaml:"importers"`
}

// NodeDependencies represents the dependencies of a Node package via its package.json and lockfile.
type NodeDependencies struct {
	PackageJSON  *PackageJSON
	LockfilePath string
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

// ReadNodeDependencies looks for a package.json and lockfile in either appDir or rootDir. The
// lockfile must either be in the same directory as package.json or be in the application root.
// TODO (b/354012293): In the future we should read the data into structs for easier manipulation.
func ReadNodeDependencies(ctx *gcp.Context, appDir string) (*NodeDependencies, error) {
	rootDir := ctx.ApplicationRoot()
	if !strings.HasPrefix(appDir, rootDir) {
		return nil, fmt.Errorf("appDir %q is not a subpath of application root %q", appDir, rootDir)
	}

	var dir string
	var pjs *PackageJSON
	var err error
	// Check appDir first for package.json file, then rootDir
	if pjs, err = ReadPackageJSONIfExists(appDir); err != nil {
		return nil, err
	}
	if pjs != nil {
		dir = appDir
	} else {
		if pjs, err = ReadPackageJSONIfExists(rootDir); err != nil {
			return nil, err
		}
		if pjs != nil {
			dir = rootDir
		} else {
			return nil, gcp.UserErrorf("package.json not found")
		}
	}

	// Try to find a lockfile from the same dir, if there is none then check the application root.
	if path := findValidLockfileInDir(dir); path != "" {
		return &NodeDependencies{pjs, path}, nil
	}

	if path := findValidLockfileInDir(rootDir); path != "" {
		return &NodeDependencies{pjs, path}, nil
	}

	return &NodeDependencies{pjs, ""}, nil
}

func findValidLockfileInDir(dir string) string {
	for _, filename := range possibleLockfileFilenames {
		if fp := filepath.Join(dir, filename); isValidLockFile(fp) {
			return fp
		}
	}
	return ""
}

// isValidLockFile validates that the lock file both exists and is not empty.
func isValidLockFile(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && info.Size() > 0
}

// HasGCPBuild returns true if the given package.json file includes a "gcp-build" script.
func HasGCPBuild(p *PackageJSON) bool {
	return HasScript(p, ScriptGCPBuild)
}

// HasApphostingBuild returns true if the given package.json file includes a "apphosting-build" script.
func HasApphostingBuild(p *PackageJSON) bool {
	return HasScript(p, ScriptApphostingBuild)
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
	if pjs == nil || pjs.Engines.Node == "" {
		return defaultVersionConstraint, nil
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
	var lockfileV6 PnpmV6Lockfile
	if err := yaml.Unmarshal(rawPackageLock, &lockfileV6); err != nil {
		return "", gcp.InternalErrorf("parsing pnpm lock file: %w", err)
	}
	if _, ok := lockfileV6.Dependencies[pkg]; ok {
		return strings.Split(lockfileV6.Dependencies[pkg].Version, "(")[0], nil
	}
	if _, ok := lockfileV6.DevDependencies[pkg]; ok {
		return strings.Split(lockfileV6.DevDependencies[pkg].Version, "(")[0], nil
	}
	var lockfileV9 PnpmV9Lockfile
	if err := yaml.Unmarshal(rawPackageLock, &lockfileV9); err != nil {
		return "", gcp.InternalErrorf("parsing pnpm lock file: %w", err)
	}
	if _, ok := lockfileV9.Importers.Dot.Dependencies[pkg]; ok {
		return strings.Split(lockfileV9.Importers.Dot.Dependencies[pkg].Version, "(")[0], nil
	}
	if _, ok := lockfileV9.Importers.Dot.DevDependencies[pkg]; ok {
		return strings.Split(lockfileV9.Importers.Dot.DevDependencies[pkg].Version, "(")[0], nil
	}
	return "", gcp.InternalErrorf("Failed to find version for package %s in pnpm lockfile", pkg)
}

func versionFromYarnLock(rawPackageLock []byte, pjs *PackageJSON, pkg string) (string, error) {
	// yarn requires custom parsing since it has a custom format
	// this logic works for both yarn classic and berry

	// Split using a more flexible regex to handle various newline characters across OSes
	dependencies := dependencyRegex.Split(string(rawPackageLock), -1)

	for _, dependency := range dependencies {
		if strings.Contains(dependency, pkg+"@") && strings.Contains(dependency, pjs.Dependencies[pkg]) {
			for _, line := range strings.Split(dependency, "\n") {
				if strings.Contains(line, "version") {
					return strings.Trim(strings.Fields(line)[1], `"`), nil
				}
			}
		}
	}
	return "", gcp.InternalErrorf("Failed to find version for package %s in yarn lockfile", pkg)
}

func versionFromNpmLock(rawPackageLock []byte, pkg string) (string, error) {
	var lockfile NpmLockfile
	if err := json.Unmarshal(rawPackageLock, &lockfile); err != nil {
		return "", gcp.InternalErrorf("parsing lock file: %w", err)
	}
	return lockfile.Packages["node_modules/"+pkg].Version, nil
}

// Version tries to get the concrete package version used based on lock file.
func Version(deps *NodeDependencies, pkg string) (string, error) {
	raw, err := os.ReadFile(deps.LockfilePath)
	if err != nil {
		return "", gcp.UserErrorf("reading file at path %s: %w", deps.LockfilePath, err)
	}
	switch {
	case strings.HasSuffix(deps.LockfilePath, "pnpm-lock.yaml"):
		return versionFromPnpmLock(raw, pkg)
	case strings.HasSuffix(deps.LockfilePath, "yarn.lock"):
		return versionFromYarnLock(raw, deps.PackageJSON, pkg)
	case strings.HasSuffix(deps.LockfilePath, "npm-shrinkwrap.json") || strings.HasSuffix(deps.LockfilePath, "package-lock.json"):
		return versionFromNpmLock(raw, pkg)
	}

	return "", gcp.UserErrorf("Failed to find version for package %s", pkg)
}

// parsePackageManager parses the packageManager field and returns the manager name and version.
// packageManagerField must have this regex (pnpm|yarn)@\d+\.\d+\.\d+(-.+)?, e.g. pnpm@9.0.0.
func parsePackageManager(packageManagerField string) (string, string, error) {
	packageManagerSplit := strings.Split(packageManagerField, "@")
	if len(packageManagerSplit) != 2 {
		return "", "", gcp.UserErrorf("parsing packageManager package.json field")
	}
	return packageManagerSplit[0], packageManagerSplit[1], nil
}

// MajorVersion returns the major version of a version string of format "major.minor.patch".
func MajorVersion(versionString string) (string, error) {
	parts := strings.Split(versionString, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid version format: %s", versionString)
	}

	return parts[0], nil
}
