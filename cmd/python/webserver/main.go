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
	"os"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	layerName = "gunicorn"
)

var (
	gunicornRegexp = regexp.MustCompile(`(?m)^gunicorn\b([^-]|$)`)
	eggRegexp      = regexp.MustCompile(`(?m)#egg=gunicorn$`)
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

	ctx.Logf("Installing gunicorn.")
	ctx.Exec([]string{"python3", "-m", "pip", "install", "--upgrade", "gunicorn", "-t", l.Path}, gcp.WithUserAttribution)

	l.SharedEnvironment.PrependPath("PYTHONPATH", l.Path)
	return nil
}

func gunicornPresentInRequirements(ctx *gcp.Context, path string) bool {
	content := ctx.ReadFile(path)
	return containsGunicorn(string(content))
}

func containsGunicorn(s string) bool {
	return gunicornRegexp.MatchString(s) || eggRegexp.MatchString(s)
}
