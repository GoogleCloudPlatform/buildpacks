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
	"path/filepath"
	"regexp"

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

// SupportsNoGoMod only returns true for Go version 1.11 and 1.13.
// These are the two GCF-supported versions that don't require a go.mod file.
func SupportsNoGoMod(ctx *gcp.Context) bool {
	v := GoVersion(ctx)

	version, err := semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go version string %q: %s", v, err))
	}

	go113OrLower := semver.MustParseRange("<1.14.0")
	return go113OrLower(version)
}

// SupportsAutoVendor returns true if both:
// + Go 1.14+ is installed.
// + go.mod contains a "go 1.14" or higher entry.
// Starting from Go 1.14, `go build` automatically detects and use a `vendor` folder
// if `go.mod` contains a `go 1.14` line.
func SupportsAutoVendor(ctx *gcp.Context) bool {
	return VersionMatches(ctx, ">=1.14.0")
}

// VersionMatches returns true if the given versionCheck
// string of format Boolean operator MAJOR.MINOR (e.g. ">=1.14") version check passes
// the semver check. This functions checks both GoModVersion and GoMod.
func VersionMatches(ctx *gcp.Context, versionCheck string) bool {
	v := GoModVersion(ctx)
	if v == "" {
		return false
	}

	version, err := semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go version string %q: %s", v, err))
	}

	goVersionMatches, err := semver.ParseRange(versionCheck)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go version string %q: %s", v, err))
	}

	if !goVersionMatches(version) {
		return false
	}

	v = GoVersion(ctx)

	version, err = semver.ParseTolerant(v)
	if err != nil {
		ctx.Exit(1, gcp.InternalErrorf("unable to parse go version string %q: %s", v, err))
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
