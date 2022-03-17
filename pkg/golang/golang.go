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
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
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
)

var (
	// goVersionRegexp is used to parse `go version`'s output.
	goVersionRegexp = regexp.MustCompile(`^go version go(\d+(\.\d+){1,2})([a-z]+\d+)? .*$`)

	// goModVersionRegexp is used to get correct declaration of Go version from go.mod file.
	goModVersionRegexp = regexp.MustCompile(`(?m)^\s*go\s+(\d+(\.\d+){1,2})\s*$`)
)

// SupportsAppEngineApis is a Go buildpack specific function that returns true if App Engine API access is enabled
func SupportsAppEngineApis(ctx *gcp.Context) (bool, error) {
	if os.Getenv(env.Runtime) == "go111" {
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

// VersionMatches checks if the installed version of Go and the version specified in go.mod match the given version range.
// The range string has the following format: https://github.com/blang/semver#ranges.
func VersionMatches(ctx *gcp.Context, versionRange string) (bool, error) {
	v, err := GoModVersion(ctx)
	if err != nil {
		return false, err
	}
	if v == "" {
		return false, nil
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
	v := readGoVersion(ctx)

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
var readGoVersion = func(ctx *gcp.Context) string {
	return ctx.Exec([]string{"go", "version"}).Stdout
}

// cleanModCache deletes the downloaded cached dependencies using `go clean -modcache`.
// The cached dependencies are written without write access and attempt
// to clear layer using ctx.ClearLayer(l) fails with permission denied errors.
// It can be overridden for testing.
var cleanModCache = func(ctx *gcp.Context) {
	ctx.Exec([]string{"go", "clean", "-modcache"})
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

	sha, err := cache.Hash(ctx, cache.WithFiles(goModPath(ctx)))
	if err != nil {
		if os.IsNotExist(err) {
			// when go.mod doesn't exist, clear any previously cached bits and return an empty layer
			l.Cache = false
			cleanModCache(ctx)
			return l, nil
		}
		return nil, err
	}
	shaStr := fmt.Sprintf("%x", sha)
	if shaStr == ctx.GetMetadata(l, goModCacheKey) {
		ctx.Logf("GOPATH layer cache hit")
		ctx.CacheHit(goPathLayerName)
		return l, nil
	}
	ctx.Debugf("go.mod SHA has changed: clearing GOPATH layer's cache")
	cleanModCache(ctx)
	ctx.SetMetadata(l, goModCacheKey, shaStr)
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
	supportsGoProxy, err := SupportsGoProxyFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking for go proxy support: %w", err)
	}
	if supportsGoProxy {
		opts = append(opts, gcp.WithEnv("GOPROXY=https://proxy.golang.org|direct"))
		return ctx.Exec(cmd, opts...), nil
	}

	result, err := ctx.ExecWithErr(cmd, opts...)
	if err == nil {
		return result, nil
	}
	ctx.Warnf("%q failed. Retrying with GOSUMDB=off GOPROXY=direct. Error: %v", strings.Join(cmd, " "), err)

	opts = append(opts, gcp.WithEnv("GOSUMDB=off", "GOPROXY=direct"))
	return ctx.Exec(cmd, opts...), nil
}
