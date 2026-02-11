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

// Package golang contains Go buildpack library code.
package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

const (
	// OutBin is the name of the final compiled binary produced by Go buildpacks.
	OutBin = "main"
	// BuildDirEnv is an environment variable that buildpacks can use to communicate the working directory to `go build`.
	BuildDirEnv = "GOOGLE_INTERNAL_BUILD_DIR"
	// The name of the layer where the GOPATH is stored
	goPathLayerName = "gopath"
	// The key used when a layers' cache is keyed off of the go mod
	goModCacheKey = "go-mod-sha"
	envGoVersion  = "GOOGLE_GO_VERSION"
)

var (
	// goVersionRegexp is used to parse `go version`'s output.
	goVersionRegexp = regexp.MustCompile(`^go version go(\d+(\.\d+){1,2})([a-z]+\d+)? .*$`)

	// goModVersionRegexp is used to get correct declaration of Go version from go.mod file.
	goModVersionRegexp = regexp.MustCompile(`(?m)^\s*go\s+(\d+(\.\d+){1,2})\s*$`)

	// goVersionsURL can be use to download a list of available, stable versions of Go.
	goVersionsURL = "https://go.dev/dl/?mode=json"

	// latestGoVersionPerStack is the latest Go version per stack to use if not specified by the user.
	latestGoVersionPerStack = map[string]string{
		runtime.Ubuntu2204: "1.25.*",
		runtime.Ubuntu2404: "1.25.*",
	}
)

// goRelease represents an entry on the go.dev downloads page.
type goRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// SupportsAppEngineApis is a Go buildpack specific function that returns true if App Engine API access is enabled
func SupportsAppEngineApis(ctx *gcp.Context) (bool, error) {
	if IsGo111Runtime() {
		return true, nil
	}

	return appengine.ApisEnabled(ctx)
}

// SupportsAutoVendor returns true if the Go version supports automatic detection of the vendor directory.
// This feature is supported by Go 1.14 and higher.
func SupportsAutoVendor(ctx *gcp.Context) (bool, error) {
	return VersionMatches(ctx, ">=1.14.0")
}

// SupportsGoProxyFallback returns true if the Go version supports fallback in GOPROXY using the pipe character.
// This feature is supported by Go 1.15 and higher.
func SupportsGoProxyFallback(ctx *gcp.Context) (bool, error) {
	return VersionMatches(ctx, ">=1.15.0")
}

// SupportsGoCleanModCache returns true if the Go version supports `go clean -modcache` without loading the packages.
// The command fails if the packages aren't available for Go 1.12 and lower.
// The feature to skip loading the packages is only supported by Go 1.13 and higher.
// More information can be found at golang.org/issue/28680 and golang.org/issue/28459.
func SupportsGoCleanModCache(ctx *gcp.Context) (bool, error) {
	return VersionMatches(ctx, ">=1.13.0")
}

// SupportsGoGet returns true if the Go version supports `go get`.
// For versions above 1.22.0+ `go get` is not supported outside of modules in legacy gopath mode.
func SupportsGoGet(ctx *gcp.Context) (bool, error) {
	v, err := RuntimeVersion(ctx)
	if err != nil {
		return false, err
	}
	if v == "" {
		return false, nil
	}
	return VersionMatches(ctx, "<1.22.0", v)
}

// SupportsVendorModificaton returns true if the Go version supports modifying vendor directory without modifying vendor/modules.txt.
// Versions 1.23.0 and later require vendored packages to be present in vendor/modules.txt to be imported.
func SupportsVendorModificaton(ctx *gcp.Context) (bool, error) {
	v, _ := RuntimeVersion(ctx)

	// If runtimeVersion is not set, it uses latest version (which is going to be >=1.23.0) which does
	// not support vendor modification without modifying vendor/modules.txt.
	return VersionMatches(ctx, "<1.23.0", v)
}

// VersionMatches checks if the installed version of Go and the version specified in go.mod match the given version range.
// The range string has the following format: https://github.com/blang/semver#ranges.
func VersionMatches(ctx *gcp.Context, versionRange string, goVersions ...string) (bool, error) {

	var v string
	var err error

	if len(goVersions) == 0 {
		v, err = GoModVersion(ctx)
		if err != nil {
			return false, err
		}
	} else {
		v = goVersions[0]
	}

	if v == "" {
		return false, nil
	}

	if isSupportedUnstableGoVersion(v) {
		// The format of Go pre-release version e.g. 1.20rc1 doesn't follow the semver rule
		// that requires a hyphen before the identifier "rc".
		if strings.Contains(v, "rc") && !strings.Contains(v, "-rc") {
			v = strings.Replace(v, "rc", "-rc", 1)
		}
	}

	version, err := semver.NewVersion(v)
	if err != nil {
		return false, gcp.InternalErrorf("unable to parse go.mod version string %q: %s", v, err)
	}

	goVersionMatches, err := semver.NewConstraint(versionRange)
	if err != nil {
		return false, gcp.InternalErrorf("unable to parse version range %q: %s", v, err)
	}

	if !goVersionMatches.Check(version) {
		return false, nil
	}

	v, err = GoVersion(ctx)
	if err != nil {
		return false, err
	}

	version, err = semver.NewVersion(v)
	if err != nil {
		return false, gcp.InternalErrorf("unable to parse Go version string %q: %s", v, err)
	}

	return goVersionMatches.Check(version), nil
}

// GoVersion reads the version of the installed Go runtime.
func GoVersion(ctx *gcp.Context) (string, error) {
	v, err := readGoVersion(ctx)
	if err != nil {
		return "", err
	}

	match := goVersionRegexp.FindStringSubmatch(v)
	if len(match) < 2 || match[1] == "" {
		return "", gcp.InternalErrorf("unable to find go version in %q", v)
	}

	return match[1], nil
}

// GoModVersion reads the version of Go from a go.mod file if present.
// If not present or if version isn't there returns an empty string.
func GoModVersion(ctx *gcp.Context) (string, error) {
	v, err := readGoMod(ctx)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	if v == "" {
		return v, nil
	}

	match := goModVersionRegexp.FindStringSubmatch(v)
	if len(match) < 2 || match[1] == "" {
		return "", nil
	}

	return match[1], nil
}

// readGoVersion returns the output of `go version`.
// It can be overridden for testing.
var readGoVersion = func(ctx *gcp.Context) (string, error) {
	result, err := ctx.Exec([]string{"go", "version"})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// cleanModCache deletes the downloaded cached dependencies using `go clean -modcache`.
// The cached dependencies are written without write access and attempt
// to clear layer using ctx.ClearLayer(l) fails with permission denied errors.
// It can be overridden for testing.
var cleanModCache = func(ctx *gcp.Context) error {
	_, err := ctx.Exec([]string{"go", "clean", "-modcache"})
	return err
}

// readGoMod reads the go.mod file if present. If not present, returns an empty string.
// It can be overridden for testing.
var readGoMod = func(ctx *gcp.Context) (string, error) {
	goModPath := goModPath(ctx)
	goModExists, err := ctx.FileExists(goModPath)
	if err != nil {
		return "", err
	}
	if !goModExists {
		return "", nil
	}
	bytes, err := ctx.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewGoWorkspaceLayer returns a new layer for `go env GOPATH` or the go workspace. The
// layer is configured for caching if possible. It only supports caching for "go mod"
// based builds.
func NewGoWorkspaceLayer(ctx *gcp.Context) (*libcnb.Layer, error) {
	l, err := ctx.Layer(goPathLayerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return nil, fmt.Errorf("creating %v layer: %w", goPathLayerName, err)
	}
	l.BuildEnvironment.Override("GOPATH", l.Path)
	l.BuildEnvironment.Override("GO111MODULE", "on")
	// Set GOPROXY to ensure no additional dependency is downloaded at built time.
	// All of them are downloaded here.
	l.BuildEnvironment.Override("GOPROXY", "off")

	shouldEnablePkgCache, err := SupportsGoCleanModCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking for go pkg cache support: %w", err)
	}
	if !shouldEnablePkgCache {
		l.Cache = false
		return l, nil
	}

	hash, cached, err := cache.HashAndCheck(ctx, l, goModCacheKey, cache.WithFiles(goModPath(ctx)))
	if err != nil {
		if os.IsNotExist(err) {
			// when go.mod doesn't exist, clear any previously cached bits and return an empty layer
			l.Cache = false
			cleanModCache(ctx)
			return l, nil
		}
		return nil, err
	}
	if cached {
		return l, nil
	}
	ctx.Debugf("go.mod SHA has changed: clearing GOPATH layer's cache")
	cleanModCache(ctx)
	cache.Add(ctx, l, goModCacheKey, hash)
	return l, nil
}

func goModPath(ctx *gcp.Context) string {
	return filepath.Join(ctx.ApplicationRoot(), "go.mod")
}

// ExecWithGoproxyFallback runs the given command with a GOPROXY fallback.
// Before Go 1.14, Go would fall back to direct only if a 404 or 410 error ocurred, for those
// versions, we explictly disable GOPROXY and try again on any error.
// For newer versions of Go, we take advantage of the "pipe" character which has the same effect.
func ExecWithGoproxyFallback(ctx *gcp.Context, cmd []string, opts ...gcp.ExecOption) (*gcp.ExecResult, error) {
	if _, present := os.LookupEnv("GOPROXY"); present {
		// If the user has explicitly set GOPROXY, we shouldn't clobber it.
		return ctx.Exec(cmd, opts...)
	}
	supportsGoProxy, err := SupportsGoProxyFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking for go proxy support: %w", err)
	}
	if supportsGoProxy {
		opts = append(opts, gcp.WithEnv("GOPROXY=https://proxy.golang.org|direct"))
		return ctx.Exec(cmd, opts...)
	}

	result, err := ctx.Exec(cmd, opts...)
	if err == nil {
		return result, nil
	}
	ctx.Warnf("%q failed. Retrying with GOSUMDB=off GOPROXY=direct. Error: %v", strings.Join(cmd, " "), err)

	opts = append(opts, gcp.WithEnv("GOSUMDB=off", "GOPROXY=direct"))
	return ctx.Exec(cmd, opts...)
}

// IsGo111Runtime returns true when the GOOGLE_RUNTIME is go111. This will be
// true when using GCF or GAE with go 1.11.
func IsGo111Runtime() bool {
	return os.Getenv(env.Runtime) == "go111"
}

// RuntimeVersion returns the runtime version for the go app.
func RuntimeVersion(ctx *gcp.Context) (string, error) {
	var version string

	switch {
	case os.Getenv(envGoVersion) != "":
		version = os.Getenv(envGoVersion)
		ctx.Logf("Using runtime version from %s: %s", envGoVersion, version)

	case os.Getenv(env.RuntimeVersion) != "":
		version = os.Getenv(env.RuntimeVersion)
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)

	default:
		os := runtime.OSForStack(ctx)
		var ok bool
		version, ok = latestGoVersionPerStack[os]
		if !ok {
			return "", gcp.UserErrorf("invalid stack for Go runtime: %q", os)
		}
		ctx.Logf("Go version not specified, using latest available Go runtime for the stack %q", os)
	}

	resolvedVersion, err := ResolveGoVersion(version)
	if err != nil {
		return "", err
	}

	return resolvedVersion, nil

}

// ResolveGoVersion finds the latest version of Go that matches the provided semver constraint.
var ResolveGoVersion = func(verConstraint string) (string, error) {
	if isSupportedUnstableGoVersion(verConstraint) || isExactGoSemver(verConstraint) {
		return verConstraint, nil
	}
	var releases []goRelease
	if err := fetch.JSON(goVersionsURL, &releases); err != nil {
		return "", gcp.InternalErrorf("fetching Go releases: %v", err)
	}
	var versions []string
	for _, r := range releases {
		if r.Stable {
			versions = append(versions, strings.TrimPrefix(r.Version, "go"))
		}
	}
	v, err := version.ResolveVersion(verConstraint, versions, version.WithoutSanitization)
	if err != nil {
		return "", gcp.UserErrorf("invalid Go version specified: %v, You can refer to %s for a list of stable Go releases.", goVersionsURL, err)
	}
	return v, nil
}

// When launching a new runtime, we need to test with RC candidate which will eventually be replaced
// by a stable candidate. Till then, we will support these unstable releases in the QA for testing.
func isSupportedUnstableGoVersion(constraint string) bool {
	if strings.Count(constraint, ".") == 1 && strings.Count(constraint, "rc") == 1 {
		return true
	}
	return false
}

// isExactGoSemver returns true if a given string is a precisely specified go version. That is, it
// is not a version constraint. The logic for this is unique for Go because new major releases do
// not include a trailing zero (e.g. go1.20).
func isExactGoSemver(constraint string) bool {
	if c := strings.Count(constraint, "."); c != 1 && c != 2 {
		// The constraint must include the major, minor, and patch segments to be exact. By default,
		// semver.NewVersion will set these to zero so we must validate this separately.
		return false
	}
	_, err := semver.NewVersion(constraint)
	return err == nil
}
