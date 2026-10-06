// Copyright 2021 Google LLC
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

// Package cloudfunctions contains buildpack library code for all GCF runtimes.
package cloudfunctions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	be "github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func getConfig(ctx *gcp.Context, runtime string, eg appstart.EntrypointGenerator) (appstart.Config, error) {
	var c appstart.Config
	if val := os.Getenv(env.Runtime); val != "" {
		ctx.Logf("Using %s: %s", env.Runtime, val)
		c.Runtime = val
	} else {
		ctx.Logf("Using runtime: %s", runtime)
		c.Runtime = runtime
	}

	ep, err := eg(ctx)
	if err != nil {
		return appstart.Config{}, fmt.Errorf("getting entrypoint: %w", err)
	}
	c.Entrypoint = *ep

	ctx.Logf("Using config %#v", c)
	return c, nil
}

// AssertFrameworkInjectionAllowed returns an error if framework injection is disabled.
func AssertFrameworkInjectionAllowed() error {
	shouldSkipFrameworkInjection, err := IsSkipFrameworkInjectionEnabled()
	if err != nil {
		return err
	}
	if shouldSkipFrameworkInjection {
		return be.Errorf(be.StatusFailedPrecondition, "Functions Framework must be set as a dependency when skipping automatic framework injection has been enabled via %s", SkipFrameworkInjection)
	}

	return nil
}

// Build serves as a common builder for Cloud Functions buildpacks.
func Build(ctx *gcp.Context, runtime string, eg appstart.EntrypointGenerator) error {
	// In a new layer's bin directory make a symlink, serve, that points to serve2.
	// The layer's bin directory will be prepended to the PATH, so executing serve
	// will in fact execute serve2.
	// TODO(mtraver) Remove this layer and symlink once serve2 replaces serve and is renamed.
	l, err := ctx.Layer("serve", gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	binDir := filepath.Join(l.Path, "bin")
	if err := ctx.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	if err := ctx.Symlink("/usr/bin/serve2", filepath.Join(l.Path, "bin", "serve")); err != nil {
		return err
	}

	c, err := getConfig(ctx, runtime, eg)
	if err != nil {
		return fmt.Errorf("building config: %w", err)
	}

	if err := c.Write(ctx); err != nil {
		return err
	}

	ctx.AddWebProcess([]string{"pid1"})
	return nil
}
