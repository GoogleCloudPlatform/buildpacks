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

// Implements /bin/build for python/appengine buildpack.
package main

import (
	"fmt"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/blang/semver"
)

const (
	command = "gunicorn -b :$PORT main:app"
)

var (
	versionRegexp = regexp.MustCompile(`version\ (.*?)\)`)
	minVersion    = semver.MustParse("19.0.0")
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	// Always opt in.
	return nil
}

func generateEndpoint(ctx *gcp.Context) (*appengine.Entrypoint, error) {
	// Check installed gunicorn version and warn if version is lower than supported
	raw := ctx.Exec([]string{"gunicorn", "--version"}).Stdout
	match := versionRegexp.FindStringSubmatch(raw)
	if len(match) < 2 || match[1] == "" {
		return nil, fmt.Errorf("unable to find gunicorn version in %q", raw)
	}

	versionString := match[1]
	version, err := semver.ParseTolerant(versionString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse gunicorn version string %q: %v", versionString, err)
	}

	if version.LT(minVersion) {
		ctx.Warnf("Installed gunicorn version %q is less than supported version %q.", version, minVersion)
	}

	return &appengine.Entrypoint{
		Type:    appengine.EntrypointGenerated.String(),
		Command: command,
	}, nil
}

func buildFn(ctx *gcp.Context) error {
	return appengine.Build(ctx, "python", generateEndpoint)
}
