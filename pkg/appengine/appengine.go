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

// Package appengine contains buildpack library code for all runtimes.
package appengine

import (
	"fmt"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	// DefaultCommand is the command used in app_start.json if no entrypoint is specified.
	DefaultCommand = "/serve"
	// DepWarning is the warning message when an app does not enable the API flag but has App Engine API dependencies.
	DepWarning = "There is a dependency on App Engine APIs, but they are not enabled in your app.yaml. Set the app_engine_apis property."
	// IndirectDepWarning is the warning message when an app does not enable the API flag but has indirect App Engine API dependencies. This is used for runtimes that can determine indirect dependencies.
	IndirectDepWarning = "There is an indirect dependency on App Engine APIs, but they are not enabled in your app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property."
	// UnusedAPIWarning is the warning message when the API flag is enabled but no API package is included.
	UnusedAPIWarning = "App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app.yaml."
)

func getEntrypoint(ctx *gcp.Context, eg appstart.EntrypointGenerator) (*appstart.Entrypoint, error) {
	if val := os.Getenv(env.Entrypoint); val != "" {
		return &appstart.Entrypoint{
			Type:    appstart.EntrypointUser.String(),
			Command: val,
		}, nil
	}
	if eg != nil {
		return eg(ctx)
	}
	return &appstart.Entrypoint{
		Type:    appstart.EntrypointDefault.String(),
		Command: DefaultCommand,
	}, nil
}

func getConfig(ctx *gcp.Context, runtime string, eg appstart.EntrypointGenerator) (appstart.Config, error) {
	var c appstart.Config
	if val := os.Getenv(env.Runtime); val != "" {
		ctx.Debugf("Using %s: %s", env.Runtime, val)
		c.Runtime = val
	} else {
		ctx.Debugf("Using runtime: %s", runtime)
		c.Runtime = runtime
	}

	ep, err := getEntrypoint(ctx, eg)
	if err != nil {
		return appstart.Config{}, fmt.Errorf("getting entrypoint: %w", err)
	}
	c.Entrypoint = *ep

	if val := os.Getenv(env.GAEMain); val != "" {
		ctx.Debugf("Using %s: %s", env.GAEMain, val)
		c.MainExecutable = val
	}
	ctx.Debugf("Using config %#v", c)
	return c, nil
}

// Build serves as a common builder for App Engine buildpacks.
func Build(ctx *gcp.Context, runtime string, eg appstart.EntrypointGenerator) error {
	c, err := getConfig(ctx, runtime, eg)
	if err != nil {
		return fmt.Errorf("building config: %w", err)
	}

	if err := c.Write(ctx); err != nil {
		return err
	}

	ctx.AddWebProcess([]string{"/start"})
	return nil
}

// ApisEnabled returns true if the application has AppEngine API support enabled in app.yaml
func ApisEnabled(ctx *gcp.Context) (bool, error) {
	val, found := os.LookupEnv(env.AppEngineAPIs)
	if !found {
		return false, nil
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false, gcp.UserErrorf("parsing %q from %s: %v", val, env.AppEngineAPIs, err)
	}
	return parsed, nil
}

// OptInTargetPlatformAE returns a DetectResult for when a buildpack is opting in because of a 'gae' value
// for 'X_GOOGLE_TARGET_PLATFORM'.
func OptInTargetPlatformAE() gcp.DetectResult {
	return gcp.OptInEnvSet(env.TargetPlatformAppEngine)
}

// OptOutTargetPlatformNotAE returns a DetectResult for when a buildpack is opting out because the value for
// 'X_GOOGLE_TARGET_PLATFORM' is not 'gae'.
func OptOutTargetPlatformNotAE() gcp.DetectResult {
	return gcp.OptOut(fmt.Sprintf("%s not set to %q", env.XGoogleTargetPlatform, env.TargetPlatformAppEngine))
}
