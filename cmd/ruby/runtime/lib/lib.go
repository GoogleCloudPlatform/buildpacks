// Copyright 2025 Google LLC
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
package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

var osNodeVersionMap = map[string]string{
	"ubuntu1804": "12.22.12",
	"ubuntu2204": "*",
}

// Rails apps using the "webpack" gem require Node.js for asset precompilation.
func getRailsNodeVersion(ctx *gcp.Context) string {
	return osNodeVersionMap[runtime.OSForStack(ctx)]
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("ruby"); result != nil {
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
	atLeastOne, err := ctx.HasAtLeastOneOutsideDependencyDirectories("*.rb")
	if err != nil {
		return nil, fmt.Errorf("finding *.rb files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .rb files found"), nil
	}
	return gcp.OptIn("found .rb files"), nil

}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	version, err := ruby.DetectVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	rl, err := ctx.Layer("ruby", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	// Rails asset precompilation needs Node.js installed. Set the version if customer has not set it.
	if os.Getenv(nodejs.EnvNodeVersion) == "" {
		railsNodeVersion := getRailsNodeVersion(ctx)
		ctx.Logf("Setting Nodejs runtime version %s: %s", nodejs.EnvNodeVersion, railsNodeVersion)
		rl.BuildEnvironment.Override(nodejs.EnvNodeVersion, railsNodeVersion)
	}

	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Ruby, version, rl)
	if err != nil {
		return err
	}

	versionInstalled, _ := runtime.ResolveVersion(ctx, runtime.Ruby, version, runtime.OSForStack(ctx))
	// Store the installed Ruby version for subsequent buildpacks (like RubyGems) that depend on it.
	rl.BuildEnvironment.Override(ruby.RubyVersionKey, versionInstalled)

	ctx.Exec([]string{"ldd", filepath.Join(rl.Path, "lib/ruby/3.1.0/x86_64-linux/psych.so")})

	// For GAE and GCF, install RubyGems and Bundler in the same layer to maintain compatibility
	// with existing builder images.
	if env.IsGAE() || env.IsGCF() {
		err = runtime.PinGemAndBundlerVersion(ctx, version, rl)
		if err != nil {
			return fmt.Errorf("updating rubygems and bundler: %w", err)
		}
	}

	// Ruby sometimes writes to local directories tmp/ and log/, so we link these to writable areas.
	localTemp := filepath.Join(ctx.ApplicationRoot(), "tmp")
	localLog := filepath.Join(ctx.ApplicationRoot(), "log")
	ctx.Logf("Removing 'tmp' and 'log' directories in user code")
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

	return nil
}
