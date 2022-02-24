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

// Implements ruby/runtime buildpack.
// The runtime buildpack installs the Ruby runtime.
package main

import (
	"fmt"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "ruby"); result != nil {
		return result, nil
	}

	gemfileExists, err := ctx.FileExists("Gemfile")
	if err != nil {
		return nil, err
	}
	if gemfileExists {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	gemsRbExists, err := ctx.FileExists("gems.rb")
	if err != nil {
		return nil, err
	}
	if gemsRbExists {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	if !ctx.HasAtLeastOne("*.rb") {
		return gcp.OptOut("no .rb files found"), nil
	}
	return gcp.OptIn("found .rb files"), nil

}

func buildFn(ctx *gcp.Context) error {
	version, err := ruby.DetectVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	rl := ctx.Layer("ruby", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Ruby, version, rl)
	if err != nil {
		return err
	}

	// Ruby sometimes writes to local directories tmp/ and log/, so we link these to writable areas.
	localTemp := filepath.Join(ctx.ApplicationRoot(), "tmp")
	localLog := filepath.Join(ctx.ApplicationRoot(), "log")
	ctx.Logf("Removing 'tmp' and 'log' directories in user code")
	ctx.RemoveAll(localTemp)
	ctx.RemoveAll(localLog)
	ctx.Symlink("/tmp", localTemp)
	ctx.Symlink("/var/log", localLog)

	return nil
}
