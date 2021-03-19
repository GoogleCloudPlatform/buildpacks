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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/blang/semver"
)

const (
	// OutBin is the name of the final compiled binary produced by Go buildpacks.
	OutBin = "main"
	// BuildDirEnv is an environment variable that buildpacks can use to communicate the working directory to `go build`.
	BuildDirEnv = "GOOGLE_INTERNAL_BUILD_DIR"
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

// SupportsNoGoMod returns true if the Go version supports deployments without a go.mod file.
// This feature is supported by Go 1.11 and 1.13 in GCF.
func SupportsNoGoMod(ctx *gcp.Context) bool {
	v := GoVersion(ctx)

	version, err := semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go version string %q: %s", v, err))
	}

	go113OrLower := semver.MustParseRange("<1.14.0")
	return go113OrLower(version)
}

// SupportsAutoVendor returns true if the Go version supports automatic detection of the vendor directory.
// This feature is supported by Go 1.14 and higher.
func SupportsAutoVendor(ctx *gcp.Context) bool {
	return VersionMatches(ctx, ">=1.14.0")
}

// SupportsGoProxyFallback returns true if the Go versioin supports fallback in GOPROXY using the pipe character.
// This feature is supported by Go 1.15 and higher.
func SupportsGoProxyFallback(ctx *gcp.Context) bool {
	return VersionMatches(ctx, ">=1.15.0")
}

// VersionMatches checks if the installed version of Go and the version specified in go.mod match the given version range.
// The range string has the following format: https://github.com/blang/semver#ranges.
func VersionMatches(ctx *gcp.Context, versionRange string) bool {
	v := GoModVersion(ctx)
	if v == "" {
		return false
	}

	version, err := semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go.mod version string %q: %s", v, err))
	}

	goVersionMatches, err := semver.ParseRange(versionRange)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse version range %q: %s", v, err))
	}

	if !goVersionMatches(version) {
		return false
	}

	v = GoVersion(ctx)

	version, err = semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse Go version string %q: %s", v, err))
	}

	return goVersionMatches(version)
}

// GoVersion reads the version of the installed Go runtime.
func GoVersion(ctx *gcp.Context) string {
	v := readGoVersion(ctx)

	match := goVersionRegexp.FindStringSubmatch(v)
	if len(match) < 2 || match[1] == "" {
		ctx.Exit(1, gcp.InternalErrorf("unable to find go version in %q", v))
	}

	return match[1]
}

// GoModVersion reads the version of Go from a go.mod file if present.
// If not present or if version isn't there returns an empty string.
func GoModVersion(ctx *gcp.Context) string {
	v := readGoMod(ctx)
	if v == "" {
		return v
	}

	match := goModVersionRegexp.FindStringSubmatch(v)
	if len(match) < 2 || match[1] == "" {
		return ""
	}

	return match[1]
}

// readGoVersion returns the output of `go version`.
// It can be overridden for testing.
var readGoVersion = func(ctx *gcp.Context) string {
	return ctx.Exec([]string{"go", "version"}).Stdout
}

// readGoMod reads the go.mod file if present. If not present, returns an empty string.
// It can be overridden for testing.
var readGoMod = func(ctx *gcp.Context) string {
	goModPath := filepath.Join(ctx.ApplicationRoot(), "go.mod")
	if !ctx.FileExists(goModPath) {
		return ""
	}

	return string(ctx.ReadFile(goModPath))
}

// ExecWithGoproxyFallback runs the given command with a GOPROXY fallback.
// Before Go 1.14, Go would fall back to direct only if a 404 or 410 error ocurred, for those
// versions, we explictly disable GOPROXY and try again on any error.
// For newer versions of Go, we take advantage of the "pipe" character which has the same effect.
func ExecWithGoproxyFallback(ctx *gcp.Context, cmd []string, opts ...gcp.ExecOption) *gcp.ExecResult {
	if SupportsGoProxyFallback(ctx) {
		opts = append(opts, gcp.WithEnv("GOPROXY=https://proxy.golang.org|direct"))
		return ctx.Exec(cmd, opts...)
	}

	result, err := ctx.ExecWithErr(cmd, opts...)
	if err == nil {
		return result
	}
	ctx.Warnf("%q failed. Retrying with GOSUMDB=off GOPROXY=direct. Error: %v", strings.Join(cmd, " "), err)

	opts = append(opts, gcp.WithEnv("GOSUMDB=off", "GOPROXY=direct"))
	return ctx.Exec(cmd, opts...)
}
