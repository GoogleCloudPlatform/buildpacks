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
)

const (
	layerName         = "gunicorn"
	gunicornVersonKey = "gunicorn_version"
)

var (
	gunicornRegexp = regexp.MustCompile(`(?m)^gunicorn\b([^-]|$)`)
	eggRegexp      = regexp.MustCompile(`(?m)#egg=gunicorn$`)
	versionRegexp  = regexp.MustCompile(`(?m)^gunicorn\ \((.*?)\)`)
)

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
	l := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)

	// Check for up to date gunicorn version
	raw := ctx.Exec([]string{"python3", "-m", "pip", "search", "gunicorn"}, gcp.WithUserAttribution).Stdout
	match := versionRegexp.FindStringSubmatch(raw)
	if len(match) < 2 || match[1] == "" {
		return fmt.Errorf("pip search returned unexpected gunicorn version %q", raw)
	}

	metaGunicornVersion := ctx.GetMetadata(l, gunicornVersonKey)
	version := match[1]
	ctx.Debugf("Current gunicorn version: %q", version)
	ctx.Debugf(" Cached gunicorn version: %q", metaGunicornVersion)
	if version == metaGunicornVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(layerName)

	if metaGunicornVersion == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}

	ctx.Logf("Installing gunicorn.")
	ctx.Exec([]string{"python3", "-m", "pip", "install", "--upgrade", "gunicorn", "-t", l.Path}, gcp.WithUserAttribution)

	l.SharedEnvironment.PrependPath("PYTHONPATH", l.Path)
	ctx.SetMetadata(l, gunicornVersonKey, version)
	return nil
}

func gunicornPresentInRequirements(ctx *gcp.Context, path string) bool {
	content := ctx.ReadFile(path)
	return containsGunicorn(string(content))
}

func containsGunicorn(s string) bool {
	return gunicornRegexp.MatchString(s) || eggRegexp.MatchString(s)
}
