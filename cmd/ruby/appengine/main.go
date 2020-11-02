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
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
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
	ctx.RemoveAll(localTemp)
	ctx.RemoveAll(localLog)
	ctx.Symlink("/tmp", localTemp)
	ctx.Symlink("/var/log", localLog)

	return appengine.Build(ctx, "ruby",
		func(ctx *gcp.Context) (*appengine.Entrypoint, error) {
			return entrypoint(ctx, ctx.ApplicationRoot())
		})
}

func entrypoint(ctx *gcp.Context, srcDir string) (*appengine.Entrypoint, error) {
	var ep string
	ctx.Logf("WARNING: No entrypoint specified. Attempting to infer entrypoint, but it is recommended to set an explicit `entrypoint` in app.yaml.")
	if ctx.FileExists(srcDir, railsIndicator) {
		ep = maybeBundle(ctx, srcDir, railsCommand)
	} else if ctx.FileExists(srcDir, rackIndicator) {
		ep = maybeBundle(ctx, srcDir, rackCommand)
	} else {
		return nil, gcp.UserErrorf("unable to infer entrypoint, please set the `entrypoint` field in app.yaml: https://cloud.google.com/appengine/docs/standard/ruby/runtime#application_startup")
	}
	ctx.Logf("Using inferred entrypoint: %q", ep)
	return &appengine.Entrypoint{
		Type:    appengine.EntrypointGenerated.String(),
		Command: ep,
	}, nil
}

func maybeBundle(ctx *gcp.Context, srcDir, cmd string) string {
	if ctx.FileExists(srcDir, bundleIndicator) || ctx.FileExists(srcDir, bundle2Indicator) {
		return "bundle exec " + cmd
	}
	return cmd
}
