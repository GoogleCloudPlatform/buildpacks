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

// Implements python/webserver buildpack.
// The webserver buildpack installs gunicorn if a custom entrypoint is not specified.
package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName string = "gunicorn"
)

var (
	gunicornRegexp = regexp.MustCompile(`(?m)^gunicorn\b([^-]|$)`)
	eggRegexp      = regexp.MustCompile(`(?m)#egg=gunicorn$`)
	versionRegexp  = regexp.MustCompile(`(?m)^gunicorn\ \((.*?)\)`)
)

// metadata represents metadata stored for a dependencies layer.
type metadata struct {
	GunicornVersion string `toml:"gunicorn_version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if os.Getenv(env.Entrypoint) != "" {
		ctx.OptOut("custom entrypoint present")
	}
	if ctx.FileExists("requirements.txt") && gunicornPresentInRequirements(ctx, "requirements.txt") {
		ctx.OptOut("gunicorn present in requirements.txt")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	var meta metadata
	l := ctx.Layer(layerName)
	ctx.ReadMetadata(l, &meta)

	// Check for up to date gunicorn version
	raw := ctx.ExecUser([]string{"python3", "-m", "pip", "search", "gunicorn"}).Stdout
	match := versionRegexp.FindStringSubmatch(raw)
	if len(match) < 2 || match[1] == "" {
		return fmt.Errorf("pip search returned unexpected gunicorn version %q", raw)
	}

	version := match[1]
	ctx.Debugf("Current gunicorn version: %q", version)
	ctx.Debugf(" Cached gunicorn version: %q", meta.GunicornVersion)
	if version == meta.GunicornVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(layerName)

	if meta.GunicornVersion == "" {
		ctx.Logf("No metadata found or no gunicorn version found on metadata. Unable to use cache layer.")
	}

	ctx.Logf("Installing gunicorn.")
	ctx.ExecUser([]string{"python3", "-m", "pip", "install", "--upgrade", "gunicorn", "-t", l.Root})

	ctx.PrependPathSharedEnv(l, "PYTHONPATH", l.Root)

	meta.GunicornVersion = version
	ctx.WriteMetadata(l, &meta, layers.Build, layers.Cache, layers.Launch)
	return nil
}

func gunicornPresentInRequirements(ctx *gcp.Context, path string) bool {
	content := ctx.ReadFile(path)
	return containsGunicorn(string(content))
}

func containsGunicorn(s string) bool {
	return gunicornRegexp.MatchString(s) || eggRegexp.MatchString(s)
}
