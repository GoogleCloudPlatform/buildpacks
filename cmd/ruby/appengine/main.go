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

// Implements ruby/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	bundleIndicator  = "Gemfile.lock"
	bundle2Indicator = "gems.locked"
	railsIndicator   = "bin/rails"
	railsCommand     = "bin/rails server"
	rackIndicator    = "config.ru"
	rackCommand      = "rackup --port $PORT"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	// Ruby sometimes writes to local directories tmp/ and log/, so we link these to writable areas.
	localTemp := filepath.Join(ctx.ApplicationRoot(), "tmp")
	localLog := filepath.Join(ctx.ApplicationRoot(), "log")
	if err := ctx.RemoveAll(localTemp); err != nil {
		return err
	}
	if err := ctx.RemoveAll(localLog); err != nil {
		return err
	}
	if err := ctx.Symlink("/tmp", localTemp); err != nil {
		return err
	}
	if err := ctx.Symlink("/var/log", localLog); err != nil {
		return err
	}

	return appengine.Build(ctx, "ruby",
		func(ctx *gcp.Context) (*appstart.Entrypoint, error) {
			return entrypoint(ctx, ctx.ApplicationRoot())
		})
}

func entrypoint(ctx *gcp.Context, srcDir string) (*appstart.Entrypoint, error) {
	var ep string
	ctx.Logf("WARNING: No entrypoint specified. Attempting to infer entrypoint, but it is recommended to set an explicit `entrypoint` in app.yaml.")
	ep, err := inferEntrypoint(ctx, srcDir)
	if err != nil {
		return nil, err
	}
	ctx.Logf("Using inferred entrypoint: %q", ep)
	return &appstart.Entrypoint{
		Type:    appstart.EntrypointGenerated.String(),
		Command: ep,
	}, nil
}

func inferEntrypoint(ctx *gcp.Context, srcDir string) (string, error) {
	indicatorCmds := map[string]string{
		railsIndicator: railsCommand,
		rackIndicator:  rackCommand,
	}
	for indc, cmd := range indicatorCmds {
		exists, err := ctx.FileExists(srcDir, indc)
		if err != nil {
			return "", err
		}
		if exists {
			return maybeBundle(ctx, srcDir, cmd)
		}
	}
	return "", gcp.UserErrorf("unable to infer entrypoint, please set the `entrypoint` field in app.yaml: https://cloud.google.com/appengine/docs/standard/ruby/runtime#application_startup")
}

func maybeBundle(ctx *gcp.Context, srcDir, cmd string) (string, error) {
	for _, indc := range []string{bundleIndicator, bundle2Indicator} {
		exists, err := ctx.FileExists(srcDir, indc)
		if err != nil {
			return "", err
		}
		if exists {
			return fmt.Sprintf("bundle exec %s", cmd), nil
		}
	}
	return cmd, nil
}
