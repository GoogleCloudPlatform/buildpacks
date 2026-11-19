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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
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

	// ApphostingPreprocessedPathForPack is the path to the preprocessed apphosting.yaml file in the workspace.
	ApphostingPreprocessedPathForPack = "/workspace/apphosting_preprocessed"

	// NPMStartEntrypointCapability is the capability key for the NPMStartEntrypoint.
	NPMStartEntrypointCapability = "nodejs.NPMStartEntrypoint"

	nodeVersionKey = "node_version"

	// DependencyHashKey is the metadata key for package dependency hashes.
	DependencyHashKey = "dependency_hash"
)

const (
	// DevDepType is the dependency type for a dev dependency.
	DevDepType = "dev"
	// OptionalDepType is the dependency type for an optional dependency.
	OptionalDepType = "optional"
	// DevOptionalDepType is the dependency type for a dev-optional dependency.
	DevOptionalDepType = "devOptional"
	// NilDepType is the dependency type for a nil dependency.
	NilDepType = ""
)

// semVer11 is the smallest possible semantic version with major version 11.
var semVer11 = semver.MustParse("11.0.0")

var (
	// latestNodejsVersionPerStack is the latest Nodejs version per stack to use if not specified by the user.
	latestNodejsVersionPerStack = map[string]string{
		runtime.Ubuntu1804: "22.*.*",
		runtime.Ubuntu2204: "22.*.*",
		runtime.Ubuntu2404: "24.*.*",
	}

	cachedPackageJSONs        = map[string]*PackageJSON{}
	possibleLockfileFilenames = []string{"pnpm-lock.yaml", "yarn.lock", "npm-shrinkwrap.json", "package-lock.json", "bun.lock", "bun.lockb"}
	dependencyRegex           = regexp.MustCompile(`\r?\n\r?\n`)
)

type packageEnginesJSON struct {
	Node string `json:"node"`
	NPM  string `json:"npm"`
	Yarn string `json:"yarn"`
	PNPM string `json:"pnpm"`
	Bun  string `json:"bun"`
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
	Main            string             `json:"main,omitempty"`
	Type            string             `json:"type,omitempty"`
	Version         string             `json:"version,omitempty"`
	Engines         packageEnginesJSON `json:"engines,omitempty"`
	Scripts         map[string]string  `json:"scripts"`
	Dependencies    map[string]string  `json:"dependencies"`
	DevDependencies map[string]string  `json:"devDependencies"`
	PackageManager  string             `json:"packageManager,omitempty"`
}

// NPMLockfile represents the structure of a package-lock.json file.
type NPMLockfile struct {
	LockfileVersion int                `json:"lockfileVersion"`
	Packages        map[string]Package `json:"packages"`
}

// Package represents a dependency in the "packages" map (npm v2/v3) in package-lock.json.
type Package struct {
	Version              string            `json:"version"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	Dev                  bool              `json:"dev"`
	Optional             bool              `json:"optional"`
	DevOptional          bool              `json:"devOptional"`
}

// DepType returns the dependency type string based on Dev, Optional, and DevOptional flags.
func (p Package) DepType() string {
	if p.DevOptional || (p.Dev && p.Optional) {
		return DevOptionalDepType
	} else if p.Dev {
		return DevDepType
	} else if p.Optional {
		return OptionalDepType
	}
	return NilDepType
}

// ReadPackageLockJSONIfExists returns deserialized package-lock.json from the given dir.
// If the provided dir does not contain a package-lock.json file, it returns nil.
func ReadPackageLockJSONIfExists(dir string) (*NPMLockfile, error) {
	f := filepath.Join(dir, "package-lock.json")
	data, err := os.ReadFile(f)
	if os.IsNotExist(err) {
		// Return nil if the file doesn't exist (null object pattern).
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading package-lock.json: %v", err)
	}

	var lockfile NPMLockfile
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, gcp.UserErrorf("unmarshalling package-lock.json: %v", err)
	}
	return &lockfile, nil
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

// HasApphostingPackageOrYamlBuild returns true if the given package.json file includes a "apphosting:build" script or if apphosting.yaml contains a build command..
func HasApphostingPackageOrYamlBuild(p *PackageJSON, apphostingSchema apphostingschema.AppHostingSchema) bool {
	return HasApphostingPackageBuild(p) || apphostingSchema.Scripts.BuildCommand != ""
}

// HasApphostingPackageBuild returns true if the given package.json file includes a "apphosting:build" script.
func HasApphostingPackageBuild(p *PackageJSON) bool {
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

// DependencyVersion returns the version of the given dependency in the given package.json file.
func DependencyVersion(p *PackageJSON, name string) string {
	if p == nil || len(p.Dependencies) == 0 {
		return ""
	}
	version := p.Dependencies[name]
	return version
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
		os := runtime.OSForStack(ctx)
		latestNodejsVersionForStack, ok := latestNodejsVersionPerStack[os]
		if !ok {
			return "", gcp.UserErrorf("invalid stack for Nodejs runtime: %q", os)
		}
		ctx.Logf("Nodejs version not specified, using the latest available Nodejs runtime for the stack %q", os)
		return latestNodejsVersionForStack, nil
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

// VersionMatchesSemver checks if the provided version matches the given version semver range.
// The range string has the following format: https://github.com/blang/semver#ranges.
func VersionMatchesSemver(ctx *gcp.Context, versionRange string, version string) (bool, error) {
	if version == "" {
		return false, nil
	}
	constraint, err := semver.NewConstraint(versionRange)
	if err != nil {
		return false, fmt.Errorf("invalid version range %q: %w", versionRange, err)
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", version, err)
	}
	if !constraint.Check(v) {
		ctx.Debugf("Nodejs version %q does not match the semver constraint %q", version, versionRange)
		return false, nil
	}
	return true, nil
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
	hash, cached, err := cache.HashAndCheck(ctx, l, DependencyHashKey, opts...)
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
	cache.Add(ctx, l, DependencyHashKey, hash)
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
		if (strings.HasPrefix(dependency, pkg+"@") ||
			// For scoped package names, which begin with '@', the yarn lockfile wraps the name in quotes.
			// e.g. "@yarnpkg/lockfile@1.1.0".
			strings.HasPrefix(dependency, fmt.Sprintf("\"%s@", pkg))) &&
			strings.Contains(dependency, pjs.Dependencies[pkg]) {
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
	var lockfile NPMLockfile
	if err := json.Unmarshal(rawPackageLock, &lockfile); err != nil {
		return "", gcp.InternalErrorf("parsing lock file: %w", err)
	}
	return lockfile.Packages["node_modules/"+pkg].Version, nil
}

// versionFromBunLock returns the resolved version of pkg from the contents of a
// text bun.lock file. bun.lock is JSONC (JSON with comments and trailing
// commas); each entry in its "packages" map is a tuple whose first element is
// "<name>@<version>". Entries hoisted under a parent (e.g. "parent/pkg") are
// also matched. It returns ("", nil) when pkg is not in the lockfile.
func versionFromBunLock(rawPackageLock []byte, pkg string) (string, error) {
	var parsed struct {
		Packages map[string]json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(stripJSONC(rawPackageLock), &parsed); err != nil {
		return "", gcp.UserErrorf("parsing %s: %w", BunLock, err)
	}
	// Look up the top-level entry first. Indexing and ranging over a nil map
	// are no-ops, so a lockfile without "packages" falls through to ("", nil).
	if ver := extractBunPackageEntryVersion(parsed.Packages[pkg], pkg); ver != "" {
		return ver, nil
	}
	// Fall back to a copy nested under a parent package, keyed "<parent>/<pkg>".
	// TODO: b/582363483 - Read only the top-level entry (like versionFromNpmLock),
	// or sort the keys so nested matches are deterministic across runs.
	suffix := "/" + pkg
	for k, entry := range parsed.Packages {
		if strings.HasSuffix(k, suffix) {
			if ver := extractBunPackageEntryVersion(entry, pkg); ver != "" {
				return ver, nil
			}
		}
	}
	return "", nil
}

// extractBunPackageEntryVersion returns the version of pkg from a single
// bun.lock "packages" entry, or "" if none is found. A tuple entry
// (["<name>@<version>", ...]) yields the text after "<pkg>@", with any "npm:"
// prefix removed; a first element that contains no "@" is returned as is. An
// object entry ({"version": "<version>"}) yields its "version" field.
func extractBunPackageEntryVersion(raw json.RawMessage, pkg string) string {
	if len(raw) == 0 {
		return ""
	}
	if ver, ok := bunTupleEntryVersion(raw, pkg); ok {
		return ver
	}
	var obj struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	return obj.Version
}

// bunTupleEntryVersion returns the version of pkg from a tuple entry
// (["<name>@<version>", ...]) and whether one was found. It reports false when
// raw is not a non-empty array whose first element is a string, or when that
// string names a different package, so the caller can try the object form.
func bunTupleEntryVersion(raw json.RawMessage, pkg string) (string, bool) {
	var tuple []json.RawMessage
	if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) == 0 {
		return "", false
	}
	var ident string
	if err := json.Unmarshal(tuple[0], &ident); err != nil || ident == "" {
		return "", false
	}
	// Scoped names work too: "@scope/pkg@1.2.3" minus "@scope/pkg@" is "1.2.3".
	if ver, ok := strings.CutPrefix(ident, pkg+"@"); ok {
		// Aliased dependencies are recorded as "<name>@npm:<version>".
		return strings.TrimPrefix(ver, "npm:"), true
	}
	if !strings.Contains(ident, "@") {
		return ident, true
	}
	return "", false
}

// stripJSONC strips line/block comments and trailing commas from JSONC input
// (such as bun.lock).
func stripJSONC(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inString := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(src) {
				i++
				out = append(out, src[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			// Skip to the end of the line, keeping the newline.
			for i += 2; i < len(src) && src[i] != '\n'; i++ {
			}
			if i < len(src) {
				out = append(out, '\n')
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			// Skip past the closing "*/".
			for i += 2; i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/'); i++ {
			}
			i++
		case c == '}' || c == ']':
			// Drop a trailing comma before this closing bracket, keeping any
			// whitespace in between.
			j := len(out) - 1
			for j >= 0 && (out[j] == ' ' || out[j] == '\t' || out[j] == '\r' || out[j] == '\n') {
				j--
			}
			if j >= 0 && out[j] == ',' {
				out = append(out[:j], out[j+1:]...)
			}
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return out
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
// packageManagerField must have this regex (pnpm|yarn|bun)@\d+\.\d+\.\d+(-.+)?, e.g. pnpm@9.0.0.
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

// MakerNPMStartEntrypoint implements the entrypoint capability for the maker tool.
type MakerNPMStartEntrypoint struct{}

// command returns the entrypoint command for the maker tool.
func (e *MakerNPMStartEntrypoint) command() []string {
	return []string{"npm", "run", "start"}
}

// Entrypoint returns the entrypoint command based on the package manager and capabilities.
func Entrypoint(ctx *gcp.Context, packageManager string) ([]string, error) {
	if cap := ctx.Capability(NPMStartEntrypointCapability); cap != nil {
		c, ok := cap.(*MakerNPMStartEntrypoint)
		if !ok {
			return nil, gcp.InternalErrorf("capability %q must be of type *MakerNPMStartEntrypoint", NPMStartEntrypointCapability)
		}
		return c.command(), nil
	}
	return []string{packageManager, "run", "start"}, nil
}

// OverrideAppHostingBuildScript overrides the "apphosting:build" script in package.json
// with the build command from the preprocessed apphosting.yaml.
func OverrideAppHostingBuildScript(ctx *gcp.Context, preprocessedApphostingPath string) (*PackageJSON, error) {
	pjs, err := ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return nil, err
	}
	apphostingSchema, err := apphostingschema.ReadAndValidateFromFile(preprocessedApphostingPath)
	if err != nil {
		return nil, err
	}
	if apphostingSchema.Scripts.BuildCommand == "" {
		return pjs, nil
	}
	if pjs == nil {
		pjs = &PackageJSON{}
	}

	if pjs.Scripts == nil {
		pjs.Scripts = make(map[string]string)
	}

	pjs.Scripts[ScriptApphostingBuild] = apphostingSchema.Scripts.BuildCommand
	marshalledJSON, err := json.Marshal(pjs)
	if err != nil {
		return nil, gcp.InternalErrorf("marshaling package.json: %w", err)
	}

	err = os.WriteFile(filepath.Join(ctx.ApplicationRoot(), "package.json"), marshalledJSON, 0644)
	if err != nil {
		return nil, gcp.InternalErrorf("writing package.json: %w", err)
	}

	return pjs, nil
}

// IsPackageManagerConfigured checks if the environment is configured to use the specified package manager.
func IsPackageManagerConfigured(pm string) bool {
	pmPreference := os.Getenv(env.PackageManager)
	return strings.EqualFold(pmPreference, pm) // Case insensitive comparison.
}

// SkipPruningDevSync returns true if dev sync is enabled, or if dev sync status cannot be determined.
func SkipPruningDevSync(ctx *gcp.Context) bool {
	devSync, err := env.IsDevSync()
	if err != nil {
		ctx.Warnf("Unable to determine dev sync status: %v", err)
		return true // If we can't determine, we skip pruning to be safe.
	}
	if devSync {
		ctx.Logf("Skipping pruning devDependencies because dev sync is enabled.")
	}
	return devSync
}

// ShouldPrunePnpmBun returns true if dev dependencies should be pruned for pnpm/bun.
func ShouldPrunePnpmBun(ctx *gcp.Context, pjs *PackageJSON, buildNodeEnv string, nodeEnvPresent bool) bool {
	if !HasDevDependencies(pjs) {
		return false
	}
	if nodeEnvPresent {
		if buildNodeEnv != EnvProduction {
			ctx.Logf("Retaining devDependencies because NODE_ENV=%q.", buildNodeEnv)
		}
		return false
	}
	if buildNodeEnv != EnvDevelopment {
		return false
	}
	if SkipPruningDevSync(ctx) {
		return false
	}
	// We don't prune if the user is using App Hosting since App Hosting builds don't
	// rely on the node_modules folder at this point.
	if env.IsFAH() {
		return false
	}
	return true
}

// IsAngularApplication returns true if angular.json exists or @angular/core is present in package.json.
func IsAngularApplication(appDir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(appDir, "angular.json")); err == nil {
		return true, nil
	}
	pjs, err := ReadPackageJSONIfExists(appDir)
	if err != nil {
		return false, err
	}
	if pjs != nil {
		if _, ok := pjs.Dependencies["@angular/core"]; ok {
			return true, nil
		}
		if _, ok := pjs.DevDependencies["@angular/core"]; ok {
			return true, nil
		}
	}
	return false, nil
}
