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

// Implements python/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package main

import (
	"fmt"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/Masterminds/semver"
)

var (
	versionRegexp = regexp.MustCompile(`(?m)^Version:\s+(.*)$`)
	minVersion    = semver.MustParse("19.0.0")
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsGAE() {
		return gcp.OptInEnvSet(env.XGoogleTargetPlatform), nil
	}
	return gcp.OptOut("Deployment environment is not GAE."), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	if err := validateAppEngineAPIs(ctx); err != nil {
		return err
	}
	return appengine.Build(ctx, "python", entrypoint)
}

func validateAppEngineAPIs(ctx *gcp.Context) error {
	supportsApis, err := appengine.ApisEnabled(ctx)
	if err != nil {
		return err
	}

	usingAppEngine, err := appEngineInDeps(ctx)
	if err != nil {
		return err
	}

	if supportsApis && !usingAppEngine {
		ctx.Warnf(appengine.UnusedAPIWarning)
	}

	if !supportsApis && usingAppEngine {
		ctx.Warnf(appengine.DepWarning)
	}

	return nil
}

func entrypoint(ctx *gcp.Context) (*appstart.Entrypoint, error) {
	// Check installed gunicorn version and warn if version is lower than supported
	result, err := ctx.Exec([]string{"python3", "-m", "pip", "show", "gunicorn"}, gcp.WithUserTimingAttribution)
	if err != nil {
		if result != nil && result.ExitCode == 1 {
			return nil, fmt.Errorf("gunicorn not installed: %s", result.Combined)
		}
		return nil, fmt.Errorf("pip show gunicorn: %v", err)
	}
	raw := result.Stdout
	match := versionRegexp.FindStringSubmatch(raw)
	if len(match) < 2 || match[1] == "" {
		return nil, fmt.Errorf("unable to find gunicorn version in %q", raw)
	}

	versionString := match[1]
	version, verr := semver.NewVersion(versionString)
	if verr != nil {
		return nil, fmt.Errorf("unable to parse gunicorn version string %q: %v", versionString, verr)
	}

	if version.LessThan(minVersion) {
		ctx.Warnf("Installed gunicorn version %q is less than supported version %q.", version, minVersion)
	}

	return &appstart.Entrypoint{
		Type:    appstart.EntrypointDefault.String(),
		Command: appengine.DefaultCommand,
	}, nil
}

func appEngineInDeps(ctx *gcp.Context) (bool, error) {
	// Check if appengine-python-standard is installed
	result, err := ctx.Exec([]string{"python3", "-m", "pip", "show", "appengine-python-standard"}, gcp.WithUserTimingAttribution)
	if err != nil {
		if result != nil && result.ExitCode == 1 {
			return false, nil
		}
		return false, fmt.Errorf("pip show appengine-python-standard: %v", err)
	}
	return true, nil
}
